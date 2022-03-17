package ssh

import (
	"fmt"
	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
	"github.com/loft-sh/devspace/helper/tunnel"
	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

var DefaultPort = 8022

func NewServer(addr string, keys []ssh.PublicKey) (*Server, error) {
	shell, err := getShell()
	if err != nil {
		return nil, err
	}

	forwardHandler := &ssh.ForwardedTCPHandler{}
	server := &Server{
		shell: shell,
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

				tunnel.LogDebugf("Declined public key")
				return false
			},
			LocalPortForwardingCallback: func(ctx ssh.Context, dhost string, dport uint32) bool {
				tunnel.LogDebugf("Accepted forward", dhost, dport)
				return true
			},
			ReversePortForwardingCallback: func(ctx ssh.Context, host string, port uint32) bool {
				tunnel.LogDebugf("attempt to bind", host, port, "granted")
				return true
			},
			ChannelHandlers: map[string]ssh.ChannelHandler{
				"direct-tcpip": ssh.DirectTCPIPHandler,
				"session":      ssh.DefaultSessionHandler,
			},
			RequestHandlers: map[string]ssh.RequestHandler{
				"tcpip-forward":        forwardHandler.HandleSSHRequest,
				"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
			},
			SubsystemHandlers: map[string]ssh.SubsystemHandler{
				"sftp": SftpHandler,
			},
		},
	}

	server.sshServer.Handler = server.handler
	return server, nil
}

type Server struct {
	shell     string
	sshServer ssh.Server
}

func setWinsize(f *os.File, w, h int) {
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
}

func getShell() (string, error) {
	// try to get a shell
	_, err := exec.LookPath("bash")
	if err != nil {
		_, err := exec.LookPath("sh")
		if err != nil {
			return "", fmt.Errorf("neither 'bash' nor 'sh' found in container. Please make sure at least one is available in the container $PATH")
		}

		return "sh", nil
	}

	return "bash", nil
}

func (s *Server) handler(sess ssh.Session) {
	cmd := s.getCommand(sess)
	if ssh.AgentRequested(sess) {
		l, err := ssh.NewAgentListener()
		if err != nil {
			exitWithError(sess, errors.Wrap(err, "start agent"))
			return
		}

		defer l.Close()
		go ssh.ForwardAgentConnections(l, sess)
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", "SSH_AUTH_SOCK", l.Addr().String()))
	}

	// start shell session
	var err error
	ptyReq, winCh, isPty := sess.Pty()
	if isPty {
		err = s.handlePTY(sess, ptyReq, winCh, cmd)
	} else {
		err = s.handleNonPTY(sess, cmd)
	}

	// exit session
	exitWithError(sess, err)
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

	err = cmd.Wait()
	if err != nil {
		return errors.Wrap(err, "waiting for command")
	}

	<-stdinDoneChan
	<-stdoutDoneChan
	return nil
}

func (s *Server) getCommand(sess ssh.Session) *exec.Cmd {
	var cmd *exec.Cmd
	if len(sess.RawCommand()) == 0 {
		cmd = exec.Command(s.shell)
	} else {
		args := []string{"-c", sess.RawCommand()}
		cmd = exec.Command(s.shell, args...)
	}

	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, sess.Environ()...)
	return cmd
}

func exitWithError(s ssh.Session, err error) {
	if err != nil {
		tunnel.LogErrorf("%v", err)
		msg := strings.TrimPrefix(err.Error(), "exec: ")
		if _, err := s.Stderr().Write([]byte(msg)); err != nil {
			tunnel.LogErrorf("failed to write error to session: %v", err)
		}
	}

	// always exit session
	err = s.Exit(exitCode(err))
	if err != nil {
		tunnel.LogErrorf("session failed to exit: %v", err)
	}
}

func SftpHandler(sess ssh.Session) {
	debugStream := ioutil.Discard
	serverOptions := []sftp.ServerOption{
		sftp.WithDebug(debugStream),
	}
	server, err := sftp.NewServer(
		sess,
		serverOptions...,
	)
	if err != nil {
		log.Printf("sftp server init error: %s\n", err)
		return
	}
	if err := server.Serve(); err == io.EOF {
		server.Close()
		fmt.Println("sftp client exited session.")
	} else if err != nil {
		fmt.Println("sftp server completed with error:", err)
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

func (s *Server) ListenAndServe() error {
	tunnel.LogInfof("Start ssh server on %s", s.sshServer.Addr)
	return s.sshServer.ListenAndServe()
}
