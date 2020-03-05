package cmd

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/command"
	"github.com/devspace-cloud/devspace/pkg/util/exit"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"mvdan.cc/sh/v3/interp"
)

// RunCmd holds the run cmd flags
type RunCmd struct {
	*flags.GlobalFlags
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
#######################################################
	`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunRun(f, cobraCmd, args)
		},
	}

	return runCmd
}

// RunRun executes the functionality "devspace run"
func (cmd *RunCmd) RunRun(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configLoader := f.NewConfigLoader(nil, log.Discard)
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
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
	err = command.ExecuteCommand(commands, args[0], args[1:])
	if err != nil {
		shellExitError, ok := err.(interp.ShellExitStatus)
		if ok {
			return &exit.ReturnCodeError{
				ExitCode: int(shellExitError),
			}
		}

		exitError, ok := err.(interp.ExitStatus)
		if ok {
			return &exit.ReturnCodeError{
				ExitCode: int(exitError),
			}
		}

		return errors.Wrap(err, "execute command")
	}

	return nil
}
