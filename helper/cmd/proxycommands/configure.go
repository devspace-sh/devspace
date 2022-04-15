package proxycommands

import (
	"encoding/base64"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devspace/helper/util/stderrlog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	sshPrivateKeyPath = "/tmp/ssh_private_key"
	sshPublicKeyPath  = "/tmp/ssh_public_key"
	proxyCommandsPath = "/tmp/proxy_commands"
)

// ConfigureCmd holds the ssh cmd flags
type ConfigureCmd struct {
	PublicKey  string
	PrivateKey string
	WorkingDir string

	GitCredentials bool

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
	configureCmd.Flags().BoolVar(&cmd.GitCredentials, "git-credentials", false, "If git credentials should get configured")
	return configureCmd
}

// Run runs the command logic
func (cmd *ConfigureCmd) Run(_ *cobra.Command, _ []string) error {
	// try to load the old commands
	oldCommands := []string{}
	out, err := ioutil.ReadFile(proxyCommandsPath)
	if err == nil {
		oldCommands = strings.Split(string(out), ",")
	}

	// first configure the commands
	for _, c := range cmd.Commands {
		filePath := "/usr/local/bin/" + c
		executeCommand := fmt.Sprintf(`#!/bin/sh
/tmp/devspacehelper proxy-commands run %s "$@"`, c)
		err := ioutil.WriteFile(filePath, []byte(executeCommand), 0777)
		if err != nil {
			return fmt.Errorf("error writing command '%s': %v", filePath, err)
		}
	}

	// remove commands that are not there anymore
	for _, oldCommand := range oldCommands {
		found := false
		for _, c := range cmd.Commands {
			if oldCommand == c {
				found = true
				break
			}
		}

		if !found {
			_ = os.Remove("/usr/local/bin/" + oldCommand)
		}
	}
	err = ioutil.WriteFile(proxyCommandsPath, []byte(strings.Join(cmd.Commands, ",")), 0644)
	if err != nil {
		stderrlog.Errorf("error writing %s: %v", proxyCommandsPath, err)
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

	// now configure git credentials
	if cmd.GitCredentials {
		homeDir, err := homedir.Dir()
		if err != nil {
			return err
		}

		gitConfigPath := filepath.Join(homeDir, ".gitconfig")
		out, err = ioutil.ReadFile(gitConfigPath)
		if err != nil || !strings.Contains(string(out), "helper = \"/tmp/devspacehelper proxy-commands git-credentials\"") {
			content := string(out) + "\n" + "[credential]" + "\n" + "        helper = \"/tmp/devspacehelper proxy-commands git-credentials\"\n"
			err = ioutil.WriteFile(gitConfigPath, []byte(content), 0644)
			if err != nil {
				return errors.Wrap(err, "write git config")
			}
		}
	}

	// print working dir to stdout
	fmt.Print(workingDir)
	return nil
}
