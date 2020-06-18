package cmd

import (
	"fmt"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/dependency"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	flagspkg "github.com/devspace-cloud/devspace/pkg/util/flags"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/sirupsen/logrus"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// RunCmd holds the run cmd flags
type RunCmd struct {
	*flags.GlobalFlags

	Dependency string
}

// NewRunCmd creates a new run command
func NewRunCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &RunCmd{GlobalFlags: globalFlags}

	runCmd := &cobra.Command{
		Use:                "run",
		DisableFlagParsing: true,
		Short:              "Run executes a predefined command",
		Long: `
#######################################################
##################### devspace run ####################
#######################################################
Run executes a predefined command from the devspace.yaml

Examples:
devspace run mycommand --myarg 123
devspace run mycommand2 1 2 3
devspace --dependency my-dependency run any-command --any-flag
#######################################################
	`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			log := f.GetLog()

			// get all flags till "run"
			index := -1
			for i, v := range os.Args {
				if v == "run" {
					index = i + 1
					break
				}
			}
			if index == -1 {
				return fmt.Errorf("error parsing command: couldn't find run in command: %v", os.Args)
			}

			// enable flag parsing
			cobraCmd.DisableFlagParsing = false

			// apply extra flags
			extraFlags, err := flagspkg.ApplyExtraFlags(cobraCmd, os.Args[:index], true)
			if err != nil {
				return err
			} else if cmd.Silent {
				log.SetLevel(logrus.FatalLevel)
			}

			if len(extraFlags) > 0 {
				log.Infof("Applying extra flags from environment: %s", strings.Join(extraFlags, " "))
			}

			return cmd.RunRun(f, cobraCmd, os.Args[index:])
		},
	}

	runCmd.Flags().StringVar(&cmd.Dependency, "dependency", "", "Run a command from a specific dependency")
	return runCmd
}

// RunRun executes the functionality "devspace run"
func (cmd *RunCmd) RunRun(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configOptions := cmd.ToConfigOptions()
	configLoader := f.NewConfigLoader(configOptions, f.GetLog())
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// check if we should execute a dependency command
	if cmd.Dependency != "" {
		config, err := configLoader.Load()
		if err != nil {
			return err
		}

		cache, err := configLoader.Generated()
		if err != nil {
			return err
		}

		mgr, err := f.NewDependencyManager(config, cache, nil, false, configOptions, f.GetLog())
		if err != nil {
			return err
		}

		return mgr.Command(dependency.CommandOptions{
			Dependencies: []string{cmd.Dependency},
			Command:      args[0],
			Args:         args[1:],
		})
	}

	// Parse commands
	commands, err := configLoader.ParseCommands()
	if err != nil {
		return err
	}

	// Save variables
	err = configLoader.SaveGenerated()
	if err != nil {
		return err
	}

	// Execute command
	return dependency.ExecuteCommand(commands, args[0], args[1:])
}
