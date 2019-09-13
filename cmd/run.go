package cmd

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/command"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/util/exit"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"mvdan.cc/sh/v3/interp"
)

// RunCmd holds the run cmd flags
type RunCmd struct {
	*flags.GlobalFlags
}

// NewRunCmd creates a new run command
func NewRunCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &RunCmd{GlobalFlags: globalFlags}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run executes a predefined command",
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
		RunE: cmd.RunRun,
	}

	return runCmd
}

// RunRun executes the functionality "devspace run"
func (cmd *RunCmd) RunRun(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	// Load config
	config, err := configutil.GetConfig(configutil.FromFlags(cmd.GlobalFlags))
	if err != nil {
		return err
	}

	// Execute command
	err = command.ExecuteCommand(config, args[0], args[1:])
	if err != nil {
		shellExitError, ok := err.(interp.ShellExitStatus)
		if ok {
			exit.Exit(int(shellExitError))
		}

		exitError, ok := err.(interp.ExitStatus)
		if ok {
			exit.Exit(int(exitError))
		}

		return errors.Wrap(err, "execute command")
	}

	return nil
}
