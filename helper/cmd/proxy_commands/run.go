package proxy_commands

import (
	"encoding/json"
	"github.com/loft-sh/devspace/helper/types"
	"github.com/loft-sh/devspace/pkg/util/terminal"
	"github.com/moby/term"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"os"
	"strings"
)

// RunCmd holds the ssh cmd flags
type RunCmd struct{}

// NewRunCmd creates a new ssh command
func NewRunCmd() *cobra.Command {
	cmd := &RunCmd{}
	runCmd := &cobra.Command{
		Use:                "run",
		Short:              "Runs a reverse command",
		DisableFlagParsing: true,
		RunE:               cmd.Run,
	}
	return runCmd
}

// Run runs the command logic
func (cmd *RunCmd) Run(_ *cobra.Command, args []string) error {
	key, err := ioutil.ReadFile(sshPrivateKeyPath)
	if err != nil {
		return errors.Wrap(err, "read private key")
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return errors.Wrap(err, "parse private key")
	}

	clientConfig := &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", "localhost:10567", clientConfig)
	if err != nil {
		return errors.Wrap(err, "dial ssh")
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return errors.Wrap(err, "new session")
	}
	defer session.Close()

	// set environment variables
	for _, v := range os.Environ() {
		splitted := strings.Split(v, "=")
		if len(splitted) < 2 {
			continue
		}

		err = session.Setenv(splitted[0], strings.Join(splitted[1:], "="))
		if err != nil {
			return errors.Wrap(err, "set session env")
		}
	}

	// check if we should use a pty
	tty, t := terminal.SetupTTY(os.Stdin, os.Stdout)
	if tty {
		info, ok := term.GetFdInfo(t.In)
		if ok {
			winSize, err := term.GetWinsize(info)
			if err == nil {
				err = session.RequestPty("xterm", int(winSize.Height), int(winSize.Width), ssh.TerminalModes{
					ssh.ECHO:          0,
					ssh.TTY_OP_ISPEED: 14400,
					ssh.TTY_OP_OSPEED: 14400,
				})
				if err != nil {
					return errors.Wrap(err, "request pty")
				}

				// TODO: terminal resize
			}
		}
	}

	// get current working directory
	currentWorkingDir, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "get current working dir")
	}

	// marshal command and execute command
	proxyCommand := &types.ProxyCommand{
		Args:       args,
		WorkingDir: currentWorkingDir,
	}
	out, err := json.Marshal(proxyCommand)
	if err != nil {
		return errors.Wrap(err, "marshal command")
	}

	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	// run the command with interrupt handlers
	err = t.Safe(func() error {
		return session.Run(string(out))
	})
	if err != nil {
		if sshExitError, ok := err.(*ssh.ExitError); ok {
			if sshExitError.ExitStatus() != 0 {
				os.Exit(sshExitError.ExitStatus())
				return nil
			}

			return nil
		}

		return err
	}

	return nil
}
