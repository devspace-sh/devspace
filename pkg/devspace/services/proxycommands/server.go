package proxycommands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gliderlabs/ssh"
	sshhelper "github.com/loft-sh/devspace/helper/ssh"
	"github.com/loft-sh/devspace/helper/types"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

func NewReverseCommandsServer(localWorkDir, containerWorkDir string, addr string, keys []ssh.PublicKey, commands []*latest.ProxyCommand, log log.Logger) *Server {
	mappings := []Mapping{
		{
			From: filepath.ToSlash(localWorkDir),
			To:   containerWorkDir,
		},
	}
	if runtime.GOOS == "windows" {
		mappings = append(mappings, Mapping{
			From: filepath.FromSlash(localWorkDir),
			To:   containerWorkDir,
		})
	}

	server := &Server{
		rewriteMappings: mappings,

		localWorkDir:     path.Clean(filepath.ToSlash(localWorkDir)),
		containerWorkDir: path.Clean(containerWorkDir),

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
			ChannelHandlers: map[string]ssh.ChannelHandler{
				"session": ssh.DefaultSessionHandler,
			},
		},
	}

	server.sshServer.Handler = server.handler
	return server
}

type Server struct {
	rewriteMappings []Mapping

	localWorkDir     string
	containerWorkDir string

	commands  []*latest.ProxyCommand
	log       log.Logger
	sshServer ssh.Server
}

func (s *Server) handler(sess ssh.Session) {
	cmd, payload, err := s.getCommand(sess)
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
	if payload.TTY && runtime.GOOS != "windows" {
		winSizeChan := make(chan ssh.Window, 1)
		winSizeChan <- ssh.Window{
			Width:  payload.Width,
			Height: payload.Height,
		}
		err = sshhelper.HandlePTY(sess, ssh.Pty{
			Term: "xterm",
			Window: ssh.Window{
				Width:  payload.Width,
				Height: payload.Height,
			},
		}, winSizeChan, cmd, func(reader io.Reader) io.Reader {
			return reader
		})
	} else {
		err = sshhelper.HandleNonPTY(sess, cmd, func(reader io.Reader) io.Reader {
			return NewRewriter(reader, s.rewriteMappings)
		})
	}

	// exit session
	s.exitWithError(sess, err)
}

func (s *Server) getCommand(sess ssh.Session) (*exec.Cmd, *types.ProxyCommand, error) {
	var cmd *exec.Cmd
	rawCommand := sess.RawCommand()
	if len(rawCommand) == 0 {
		return nil, nil, fmt.Errorf("command required")
	}

	command := &types.ProxyCommand{}
	err := json.Unmarshal([]byte(rawCommand), &command)
	if err != nil {
		return nil, nil, fmt.Errorf("parse command: %v", err)
	} else if len(command.Args) == 0 {
		return nil, nil, fmt.Errorf("command is empty")
	}

	var reverseCommand *latest.ProxyCommand
	for _, r := range s.commands {
		if r.GitCredentials && command.Args[0] == "git-credentials" {
			reverseCommand = r
			break
		}
		if r.Command == command.Args[0] {
			reverseCommand = r
			break
		}
	}
	if reverseCommand == nil {
		return nil, nil, fmt.Errorf("command not allowed")
	}

	c := reverseCommand.Command
	if reverseCommand.LocalCommand != "" {
		c = reverseCommand.LocalCommand
	}
	if reverseCommand.GitCredentials {
		c = "git"
	}

	args := []string{}
	for _, arg := range command.Args[1:] {
		splitted := strings.Split(arg, "=")
		if len(splitted) == 1 {
			args = append(args, s.transformPath(arg))
			continue
		}

		args = append(args, splitted[0]+"="+s.transformPath(strings.Join(splitted[1:], "=")))
	}

	cmd = exec.Command(c, args...)
	cmd.Dir = s.transformPath(command.WorkingDir)

	// make sure working dir exists otherwise we get an error
	_, err = os.Stat(cmd.Dir)
	if err != nil {
		s.log.Debugf("unknown working dir: %s", cmd.Dir)
		cmd.Dir = os.TempDir()
	}

	s.log.Debugf("run command '%s %s' locally", c, strings.Join(args, " "))
	if !reverseCommand.SkipContainerEnv && !reverseCommand.GitCredentials {
		cmd.Env = append(cmd.Env, command.Env...)
	}
	cmd.Env = append(cmd.Env, os.Environ()...)
	for k, v := range reverseCommand.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	if reverseCommand.GitCredentials {
		cmd.Env = append(cmd.Env, "GIT_ASKPASS=true")
	}
	return cmd, command, nil
}

func (s *Server) transformPath(originalPath string) string {
	if path.IsAbs(originalPath) {
		if originalPath == s.containerWorkDir {
			return s.localWorkDir
		} else if s.containerWorkDir == "/" {
			return path.Join(s.localWorkDir, originalPath[1:])
		} else if strings.HasPrefix(originalPath, s.containerWorkDir+"/") {
			return path.Join(s.localWorkDir, strings.TrimPrefix(originalPath, s.containerWorkDir+"/"))
		}

		relativePath, err := rel(s.containerWorkDir, originalPath)
		if err == nil {
			return path.Join(s.localWorkDir, relativePath)
		}

		// fallback to temporary folder
		return os.TempDir()
	}

	return originalPath
}

func rel(basepath, targpath string) (string, error) {
	base := path.Clean(basepath)
	targ := path.Clean(targpath)
	if targ == base {
		return ".", nil
	}
	if base == "." {
		base = ""
	}

	// Can't use IsAbs - `\a` and `a` are both relative in Windows.
	baseSlashed := len(base) > 0 && base[0] == '/'
	targSlashed := len(targ) > 0 && targ[0] == '/'
	if baseSlashed != targSlashed {
		return "", errors.New("Rel: can't make " + targpath + " relative to " + basepath)
	}
	// Position base[b0:bi] and targ[t0:ti] at the first differing elements.
	bl := len(base)
	tl := len(targ)
	var b0, bi, t0, ti int
	for {
		for bi < bl && base[bi] != '/' {
			bi++
		}
		for ti < tl && targ[ti] != '/' {
			ti++
		}
		if targ[t0:ti] != base[b0:bi] {
			break
		}
		if bi < bl {
			bi++
		}
		if ti < tl {
			ti++
		}
		b0 = bi
		t0 = ti
	}
	if base[b0:bi] == ".." {
		return "", errors.New("Rel: can't make " + targpath + " relative to " + basepath)
	}
	if b0 != bl {
		// Base elements left. Must go up before going down.
		seps := strings.Count(base[b0:bl], string('/'))
		size := 2 + seps*3
		if tl != t0 {
			size += 1 + tl - t0
		}
		buf := make([]byte, size)
		n := copy(buf, "..")
		for i := 0; i < seps; i++ {
			buf[n] = '/'
			copy(buf[n+1:], "..")
			n += 3
		}
		if t0 != tl {
			buf[n] = '/'
			copy(buf[n+1:], targ[t0:])
		}
		return string(buf), nil
	}
	return targ[t0:], nil
}

func (s *Server) exitWithError(sess ssh.Session, err error) {
	if err != nil {
		causeErr := errors.Cause(err)
		if causeErr != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				_, _ = sess.Stderr().Write([]byte(err.Error() + "\n"))
			}
		}

		s.log.Debugf("%v", err)
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
