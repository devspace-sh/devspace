package proxycommands

import (
	"encoding/json"
	"os"

	"github.com/loft-sh/devspace/helper/types"
	"github.com/loft-sh/devspace/pkg/util/terminal"
	"github.com/moby/term"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
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
	return runProxyCommand(args)
}

func runProxyCommand(args []string) error {
	key, err := os.ReadFile(sshPrivateKeyPath)
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

	// get port
	port, err := os.ReadFile(portPath)
	if err != nil {
		return errors.Wrap(err, "read port")
	}

	// dial ssh
	client, err := ssh.Dial("tcp", "localhost:"+string(port), clientConfig)
	if err != nil {
		return errors.Wrap(err, "dial ssh")
	}
	defer client.Close()

	// create new session
	session, err := client.NewSession()
	if err != nil {
		return errors.Wrap(err, "new session")
	}
	defer session.Close()

	// check if we should use a pty
	var (
		width  = 0
		height = 0
	)

	tty, t := terminal.SetupTTY(os.Stdin, os.Stdout)
	if tty {
		info, ok := term.GetFdInfo(t.In)
		if ok {
			winSize, err := term.GetWinsize(info)
			if err == nil {
				width = int(winSize.Width)
				height = int(winSize.Height)
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
		TTY:    tty,
		Width:  width,
		Height: height,

		Env:        os.Environ(),
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
