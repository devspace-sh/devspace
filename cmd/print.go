package cmd

import (
	"path/filepath"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	logger "github.com/devspace-cloud/devspace/pkg/util/log"
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
		Short: "Print displays the configuration",
		Long: `
#######################################################
################## devspace print #####################
#######################################################
Prints the configuration for the current or given 
profile after all patching and variable substitution
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
	} else if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Load config
	loadedConfig, err := configLoader.Load()
	if err != nil {
		return err
	}

	bsConfig, err := yaml.Marshal(loadedConfig)
	if err != nil {
		return err
	}

	if !cmd.SkipInfo {
		err = printExtraInfo(configLoader, log)
		if err != nil {
			return err
		}
	}

	log.WriteString(string(bsConfig))
	return nil
}

func printExtraInfo(configLoader loader.ConfigLoader, log logger.Logger) error {
	absPath, err := filepath.Abs(configLoader.ConfigPath())
	if err != nil {
		return err
	}

	log.WriteString("\n-------------------\n\nVars:\n")

	headerColumnNames := []string{"Name", "Value"}
	values := [][]string{}
	resolvedVars := configLoader.ResolvedVars()
	for varName, varValue := range resolvedVars {
		values = append(values, []string{
			varName,
			varValue,
		})
	}

	if len(values) > 0 {
		logger.PrintTable(log, headerColumnNames, values)
	} else {
		log.Info("No vars found")
	}

	log.WriteString("\n-------------------\n\nLoaded path: " + absPath + "\n\n-------------------\n\n")

	return nil
}
