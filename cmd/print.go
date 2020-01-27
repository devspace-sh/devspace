package cmd

import (
	"os"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"

	"github.com/spf13/cobra"
)

// PrintCmd is a struct that defines a command call for "print"
type PrintCmd struct {
	*flags.GlobalFlags

	SkipInfo bool
}

// NewPrintCmd creates a new devspace print command
func NewPrintCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &PrintCmd{GlobalFlags: globalFlags}

	printCmd := &cobra.Command{
		Use:   "print",
		Short: "Print builds all defined images and shows the yamls that would be deployed",
		Long: `
#######################################################
################## devspace print #####################
#######################################################
Builds all defined images and shows the yamls that would
be deployed via helm and kubectl, but skips actual 
deployment.
#######################################################`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	printCmd.Flags().BoolVar(&cmd.SkipInfo, "skip-info", false, "When enabled, only prints the configuration without additional information")

	return printCmd
}

// Run executes the command logic
func (cmd *PrintCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	log := f.GetLog()
	configOptions := cmd.ToConfigOptions()
	configLoader := f.NewConfigLoader(configOptions, log)
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Load config
	loadedConfig, err := configLoader.Load()
	if err != nil {
		return err
	}

	bs, err := yaml.Marshal(loadedConfig)
	if err != nil {
		return err
	}

	path := constants.DefaultConfigPath
	if configOptions.ConfigPath != "" {
		path = configOptions.ConfigPath
	}

	os.Stdout.Write([]byte("Loaded path: " + path + "\n-------------------\n"))
	os.Stdout.Write(bs)

	return nil
}
