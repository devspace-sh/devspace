package reversecommands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

func NewReverseCommandsServer(addr string, keys []ssh.PublicKey, commands map[string]*latest.ReverseCommand, log log.Logger) *Server {
	server := &Server{
		commands: commands,
		log:      log,
		sshServer: ssh.Server{
			Addr: addr,
			PublicKeyHandler: func(ctx ssh.Context, key ssh.PublicKey) bool {
				if len(keys) == 0 {
					return true
				}

				for _, k := range keys {
					if ssh.KeysEqual(k, key) {
						return true
					}
				}

				log.Debugf("Declined public key")
				return false
			},
			LocalPortForwardingCallback: func(ctx ssh.Context, dhost string, dport uint32) bool {
				return false
			},
			ReversePortForwardingCallback: func(ctx ssh.Context, host string, port uint32) bool {
				return false
			},
			ChannelHandlers: map[string]ssh.ChannelHandler{
				"session": ssh.DefaultSessionHandler,
			},
		},
	}

	server.sshServer.Handler = server.handler
	return server
}

type Server struct {
	commands  map[string]*latest.ReverseCommand
	log       log.Logger
	sshServer ssh.Server
}

func setWinsize(f *os.File, w, h int) {
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
}

func (s *Server) handler(sess ssh.Session) {
	cmd, err := s.getCommand(sess)
	if err != nil {
		s.exitWithError(sess, errors.Wrap(err, "construct command"))
		return
	}

	if ssh.AgentRequested(sess) {
		l, err := ssh.NewAgentListener()
		if err != nil {
			s.exitWithError(sess, errors.Wrap(err, "start agent"))
			return
		}

		defer l.Close()
		go ssh.ForwardAgentConnections(l, sess)
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", "SSH_AUTH_SOCK", l.Addr().String()))
	}

	// start shell session
	ptyReq, winCh, isPty := sess.Pty()
	if isPty && runtime.GOOS != "windows" {
		err = s.handlePTY(sess, ptyReq, winCh, cmd)
	} else {
		err = s.handleNonPTY(sess, cmd)
	}

	// exit session
	s.exitWithError(sess, err)
}

func (s *Server) handleNonPTY(sess ssh.Session, cmd *exec.Cmd) (err error) {
	stdoutReader, stdoutWriter := io.Pipe()
	stdinReader, stdinWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()
	defer stdoutWriter.Close()
	defer stdinReader.Close()
	defer stderrReader.Close()

	cmd.Stdin = stdinReader
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter
	err = cmd.Start()
	if err != nil {
		return errors.Wrap(err, "start command")
	}

	go func() {
		defer stdoutReader.Close()

		_, _ = io.Copy(sess, stdoutReader)
	}()

	go func() {
		defer stderrReader.Close()

		_, _ = io.Copy(sess.Stderr(), stderrReader)
	}()

	go func() {
		defer stdinWriter.Close()

		_, _ = io.Copy(stdinWriter, sess)
	}()

	ctx := sess.Context()
	if done := ctx.Done(); done != nil {
		go func() {
			<-done
			_ = cmd.Process.Signal(os.Kill)
		}()
	}

	err = cmd.Wait()
	if err != nil {
		return errors.Wrap(err, "wait for command")
	}

	return nil
}

func (s *Server) handlePTY(sess ssh.Session, ptyReq ssh.Pty, winCh <-chan ssh.Window, cmd *exec.Cmd) (err error) {
	cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
	f, err := pty.Start(cmd)
	if err != nil {
		return errors.Wrap(err, "start pty")
	}
	defer f.Close()

	go func() {
		for win := range winCh {
			setWinsize(f, win.Width, win.Height)
		}
	}()

	stdinDoneChan := make(chan struct{})
	go func() {
		defer f.Close()
		defer close(stdinDoneChan)

		// copy stdin
		_, _ = io.Copy(f, sess)
	}()

	stdoutDoneChan := make(chan struct{})
	go func() {
		defer f.Close()
		defer close(stdoutDoneChan)

		// copy stdout
		_, _ = io.Copy(sess, f)
	}()

	ctx := sess.Context()
	if done := ctx.Done(); done != nil {
		go func() {
			<-done
			_ = cmd.Process.Signal(os.Kill)
		}()
	}

	err = cmd.Wait()
	if err != nil {
		return errors.Wrap(err, "waiting for command")
	}

	select {
	case <-stdinDoneChan:
	case <-stdoutDoneChan:
	case <-time.After(time.Second):
	}
	return nil
}

func (s *Server) getCommand(sess ssh.Session) (*exec.Cmd, error) {
	var cmd *exec.Cmd
	rawCommand := sess.RawCommand()
	if len(rawCommand) == 0 {
		return nil, fmt.Errorf("command required")
	}

	command := []string{}
	err := json.Unmarshal([]byte(rawCommand), &command)
	if err != nil {
		return nil, fmt.Errorf("parse command: %v", err)
	}

	if s.commands[command[0]] == nil {
		return nil, fmt.Errorf("command not allowed")
	}

	c := s.commands[command[0]]
	cmd = exec.Command(c.Command, command[1:]...)
	s.log.Debugf("run command '%s %s' locally", c.Command, strings.Join(command[1:], " "))
	cmd.Env = append(cmd.Env, sess.Environ()...)
	cmd.Env = append(cmd.Env, os.Environ()...)
	return cmd, nil
}

func (s *Server) exitWithError(sess ssh.Session, err error) {
	if err != nil {
		s.log.Debugf("%v", err)
		msg := strings.TrimPrefix(err.Error(), "exec: ")
		if _, err := sess.Stderr().Write([]byte(msg)); err != nil {
			s.log.Debugf("failed to write error to session: %v", err)
		}
	}

	// always exit session
	err = sess.Exit(exitCode(err))
	if err != nil {
		s.log.Debugf("session failed to exit: %v", err)
	}
}

func exitCode(err error) int {
	err = errors.Cause(err)
	if err == nil {
		return 0
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return 1
	}

	waitStatus, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		if !exitErr.Success() {
			return 1
		}

		return 0
	}

	return waitStatus.ExitStatus()
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		_ = s.sshServer.Close()
	}()

	return s.sshServer.ListenAndServe()
}
