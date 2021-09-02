package cmd

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"io"
	"os"
	"path/filepath"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/util/factory"
	logger "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"

	"github.com/spf13/cobra"
)

// PrintCmd is a struct that defines a command call for "print"
type PrintCmd struct {
	*flags.GlobalFlags

	Out      io.Writer
	SkipInfo bool
}

// NewPrintCmd creates a new devspace print command
func NewPrintCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &PrintCmd{
		GlobalFlags: globalFlags,
		Out:         os.Stdout,
	}

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
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f)
		},
	}

	printCmd.Flags().BoolVar(&cmd.SkipInfo, "skip-info", false, "When enabled, only prints the configuration without additional information")

	return printCmd
}

// Run executes the command logic
func (cmd *PrintCmd) Run(f factory.Factory) error {
	// Set config root
	log := f.GetLog()
	configOptions := cmd.ToConfigOptions(log)
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return err
	} else if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// create kubectl client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		log.Warnf("Unable to create new kubectl client: %v", err)
	}
	configOptions.KubeClient = client

	// load config
	loadedConfig, err := configLoader.Load(configOptions, log)
	if err != nil {
		return err
	}

	// resolve dependencies
	_, err = dependency.NewManager(loadedConfig, client, configOptions, log).ResolveAll(dependency.ResolveOptions{
		Silent: true,
	})
	if err != nil {
		log.Warnf("Error resolving dependencies: %v", err)
	}

	// execute plugin hook
	err = plugin.ExecutePluginHook("print")
	if err != nil {
		return err
	}

	bsConfig, err := yaml.Marshal(loadedConfig.Config())
	if err != nil {
		return err
	}

	if !cmd.SkipInfo {
		err = printExtraInfo(cmd.ConfigPath, loadedConfig, log)
		if err != nil {
			return err
		}
	}

	if cmd.Out != nil {
		_, err := cmd.Out.Write(bsConfig)
		if err != nil {
			return err
		}
	} else {
		log.WriteString(string(bsConfig))
	}

	return nil
}

func printExtraInfo(configPath string, config config.Config, log logger.Logger) error {
	absPath, err := filepath.Abs(loader.ConfigPath(configPath))
	if err != nil {
		return err
	}

	log.WriteString("\n-------------------\n\nVars:\n")

	headerColumnNames := []string{"Name", "Value"}
	values := [][]string{}
	resolvedVars := config.Variables()
	for varName, varValue := range resolvedVars {
		values = append(values, []string{
			varName,
			fmt.Sprintf("%v", varValue),
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
