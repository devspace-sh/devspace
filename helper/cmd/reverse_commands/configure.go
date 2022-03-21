package reverse_commands

import (
	"encoding/base64"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
)

var (
	sshPrivateKeyPath = "/tmp/ssh_private_key"
	sshPublicKeyPath  = "/tmp/ssh_public_key"
	workingDirPath    = "/tmp/ssh_working_dir"
)

// ConfigureCmd holds the ssh cmd flags
type ConfigureCmd struct {
	PublicKey  string
	PrivateKey string
	WorkingDir string

	Commands []string
}

// NewConfigureCmd creates a new ssh command
func NewConfigureCmd() *cobra.Command {
	cmd := &ConfigureCmd{}
	configureCmd := &cobra.Command{
		Use:   "configure",
		Short: "Configures the remote commands in the container",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}

	configureCmd.Flags().StringVar(&cmd.PublicKey, "public-key", "", "Public key to use")
	configureCmd.Flags().StringVar(&cmd.PrivateKey, "private-key", "", "Private key to use")
	configureCmd.Flags().StringVar(&cmd.WorkingDir, "working-dir", "", "Working dir to use")
	configureCmd.Flags().StringSliceVar(&cmd.Commands, "commands", []string{}, "Commands to overwrite")
	return configureCmd
}

// Run runs the command logic
func (cmd *ConfigureCmd) Run(_ *cobra.Command, _ []string) error {
	// first configure the commands
	for _, c := range cmd.Commands {
		filePath := "/usr/local/bin/" + c
		executeCommand := fmt.Sprintf("/tmp/devspacehelper reverse-commands run %s $@", c)
		err := ioutil.WriteFile(filePath, []byte(executeCommand), 0777)
		if err != nil {
			return fmt.Errorf("error writing command '%s': %v", filePath, err)
		}
	}

	// now configure the ssh config
	if cmd.PublicKey != "" && cmd.PrivateKey != "" {
		// decode public key
		decodedPublicKey, err := base64.StdEncoding.DecodeString(cmd.PublicKey)
		if err != nil {
			return errors.Wrap(err, "decode public key")
		}

		err = ioutil.WriteFile(sshPublicKeyPath, decodedPublicKey, 0644)
		if err != nil {
			return errors.Wrap(err, "write public key")
		}

		// decode private key
		decodedPrivateKey, err := base64.StdEncoding.DecodeString(cmd.PrivateKey)
		if err != nil {
			return errors.Wrap(err, "decode private key")
		}

		err = ioutil.WriteFile(sshPrivateKeyPath, decodedPrivateKey, 0600)
		if err != nil {
			return errors.Wrap(err, "write private key")
		}
	}

	// now configure working dir
	workingDir := cmd.WorkingDir
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	err := ioutil.WriteFile(workingDirPath, []byte(workingDir), 0666)
	if err != nil {
		return errors.Wrap(err, "write working dir")
	}

	return nil
}
