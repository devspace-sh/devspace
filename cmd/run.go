package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/hook"

	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/util/factory"
	flagspkg "github.com/loft-sh/devspace/pkg/util/flags"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// RunCmd holds the run cmd flags
type RunCmd struct {
	*flags.GlobalFlags

	Dependency string
	Stdout     io.Writer
	Stderr     io.Writer
}

// NewRunCmd creates a new run command
func NewRunCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &RunCmd{
		GlobalFlags: globalFlags,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
	}

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
devspace --dependency my-dependency run any-command --any-command-flag
#######################################################
	`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
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

			// check if is help command
			osArgs := os.Args[:index]
			if len(os.Args) == index+1 && (os.Args[index] == "-h" || os.Args[index] == "--help") {
				return cobraCmd.Help()
			}

			// enable flag parsing
			cobraCmd.DisableFlagParsing = false

			// apply extra flags
			_, err := flagspkg.ApplyExtraFlags(cobraCmd, osArgs, true)
			if err != nil {
				return err
			} else if cmd.Silent {
				log.SetLevel(logrus.FatalLevel)
			}

			args := os.Args[index:]
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.RunRun(f, args)
		},
	}

	logger := f.GetLog()
	commands, _ := getCommands(f, logger)
	for _, command := range commands {
		description := command.Description
		if description == "" {
			description = "Runs command: " + command.Command
		}
		if len(description) > 64 {
			if len(description) > 64 {
				description = description[:61] + "..."
			}
		}
		runCmd.AddCommand(&cobra.Command{
			Use:                command.Name,
			Short:              description,
			Long:               description,
			Args:               cobra.ArbitraryArgs,
			DisableFlagParsing: true,
			RunE: func(cobraCmd *cobra.Command, args []string) error {
				return cobraCmd.Parent().RunE(cobraCmd, args)
			},
		})
	}

	runCmd.Flags().StringVar(&cmd.Dependency, "dependency", "", "Run a command from a specific dependency")
	return runCmd
}

// RunRun executes the functionality "devspace run"
func (cmd *RunCmd) RunRun(f factory.Factory, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("run requires at least one argument")
	}

	// Set config root
	configOptions := cmd.ToConfigOptions(f.GetLog())
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(f.GetLog())
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// check if dependency command
	commandSplitted := strings.Split(args[0], ".")
	if len(commandSplitted) > 1 {
		cmd.Dependency = strings.Join(commandSplitted[:len(commandSplitted)-1], ".")
		args[0] = commandSplitted[len(commandSplitted)-1]
	}

	// Execute plugin hook
	err = hook.ExecuteHooks(nil, nil, nil, nil, nil, "run")
	if err != nil {
		return err
	}

	// check if we should execute a dependency command
	if cmd.Dependency != "" {
		config, err := configLoader.Load(configOptions, f.GetLog())
		if err != nil {
			return err
		}

		dependencies, err := f.NewDependencyManager(config, nil, configOptions, f.GetLog()).ResolveAll(dependency.ResolveOptions{
			Silent: true,
		})
		if err != nil {
			return err
		}

		dep := dependency.GetDependencyByPath(dependencies, cmd.Dependency)
		if dep == nil {
			return fmt.Errorf("couldn't find dependency %s", cmd.Dependency)
		}

		return dep.Command(args[0], args[1:])
	}

	// load generated
	generatedConfig, err := configLoader.LoadGenerated(configOptions)
	if err != nil {
		return err
	}
	configOptions.GeneratedConfig = generatedConfig

	// Parse commands
	commandsInterface, err := configLoader.LoadWithParser(loader.NewCommandsParser(), configOptions, f.GetLog())
	if err != nil {
		return err
	}
	commands := commandsInterface.Config().Commands

	// Save variables
	err = configLoader.SaveGenerated(generatedConfig)
	if err != nil {
		return err
	}

	// Execute command
	return dependency.ExecuteCommand(commands, args[0], args[1:], cmd.Stdout, cmd.Stderr)
}

func getCommands(f factory.Factory, logger log.Logger) ([]*latest.CommandConfig, error) {
	// Set config root
	configLoader := f.NewConfigLoader("")
	configExists, err := configLoader.SetDevSpaceRoot(logger)
	if err != nil {
		return nil, err
	}
	if !configExists {
		return nil, errors.New(message.ConfigNotFound)
	}

	// Parse commands
	commandsInterface, err := configLoader.LoadWithParser(loader.NewCommandsParser(), nil, logger)
	if err != nil {
		return nil, err
	}
	return commandsInterface.Config().Commands, nil
}
