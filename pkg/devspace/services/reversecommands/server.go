package reversecommands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gliderlabs/ssh"
	sshhelper "github.com/loft-sh/devspace/helper/ssh"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func NewReverseCommandsServer(workingDir string, addr string, keys []ssh.PublicKey, commands []*latest.ReverseCommand, log log.Logger) *Server {
	server := &Server{
		workingDir: workingDir,
		commands:   commands,
		log:        log,
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
	workingDir string
	commands   []*latest.ReverseCommand
	log        log.Logger
	sshServer  ssh.Server
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
		err = sshhelper.HandlePTY(sess, ptyReq, winCh, cmd)
	} else {
		err = sshhelper.HandleNonPTY(sess, cmd)
	}

	// exit session
	s.exitWithError(sess, err)
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

	var reverseCommand *latest.ReverseCommand
	for _, r := range s.commands {
		if r.Name == command[0] {
			reverseCommand = r
			break
		}

	}
	if reverseCommand == nil {
		return nil, fmt.Errorf("command not allowed")
	}

	c := reverseCommand.Name
	if reverseCommand.Command != "" {
		c = reverseCommand.Command
	}

	cmd = exec.Command(c, command[1:]...)
	cmd.Dir = s.workingDir
	s.log.Debugf("run command '%s %s' locally", c, strings.Join(command[1:], " "))
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
	err = sess.Exit(sshhelper.ExitCode(err))
	if err != nil {
		s.log.Debugf("session failed to exit: %v", err)
	}
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		_ = s.sshServer.Close()
	}()

	return s.sshServer.ListenAndServe()
}
