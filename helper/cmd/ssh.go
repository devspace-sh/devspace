package cmd

import (
	"encoding/base64"
	"fmt"
	"github.com/gliderlabs/ssh"
	helperssh "github.com/loft-sh/devspace/helper/ssh"
	"github.com/loft-sh/devspace/helper/util/port"
	"github.com/loft-sh/devspace/helper/util/stderrlog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// SSHCmd holds the ssh cmd flags
type SSHCmd struct {
	HostKey        string
	AuthorizedKeys string
	Address        string
}

// NewSSHCmd creates a new ssh command
func NewSSHCmd() *cobra.Command {
	cmd := &SSHCmd{}
	sshCmd := &cobra.Command{
		Use:   "ssh",
		Short: "Starts a new ssh server",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}

	sshCmd.Flags().StringVar(&cmd.Address, "address", fmt.Sprintf(":%d", helperssh.DefaultPort), "Address to listen to")
	sshCmd.Flags().StringVar(&cmd.HostKey, "host-key", "", "Base64 encoded host key to use")
	sshCmd.Flags().StringVar(&cmd.AuthorizedKeys, "authorized-key", "", "Base64 encoded authorized keys to use")
	return sshCmd
}

// Run runs the command logic
func (cmd *SSHCmd) Run(_ *cobra.Command, _ []string) error {
	var keys []ssh.PublicKey
	if cmd.AuthorizedKeys != "" {
		keyBytes, err := base64.StdEncoding.DecodeString(cmd.AuthorizedKeys)
		if err != nil {
			return fmt.Errorf("seems like the provided encoded string is not base64 encoded")
		}

		for len(keyBytes) > 0 {
			key, _, _, rest, err := ssh.ParseAuthorizedKey(keyBytes)
			if err != nil {
				return errors.Wrap(err, "parse authorized key")
			}

			keys = append(keys, key)
			keyBytes = rest
		}
	}

	hostKey := []byte{}
	if len(cmd.HostKey) > 0 {
		var err error
		hostKey, err = base64.StdEncoding.DecodeString(cmd.HostKey)
		if err != nil {
			return fmt.Errorf("decode host key")
		}
	}

	server, err := helperssh.NewServer(cmd.Address, hostKey, keys)
	if err != nil {
		return err
	}

	// check if ssh is already running at that port
	available, err := port.IsAvailable(cmd.Address)
	if !available {
		if err != nil {
			return fmt.Errorf("address %s already in use: %v", cmd.Address, err)
		}

		stderrlog.Debugf("address %s already in use", cmd.Address)
		return nil
	}

	return server.ListenAndServe()
}
