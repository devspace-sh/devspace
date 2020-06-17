package cmd

import (
	"fmt"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/command"
	"github.com/devspace-cloud/devspace/pkg/util/exit"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	flagspkg "github.com/devspace-cloud/devspace/pkg/util/flags"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"mvdan.cc/sh/v3/interp"
)

// RunCmd holds the run cmd flags
type RunCmd struct {
	*flags.GlobalFlags
}

var isFlag = regexp.MustCompile("^--?[a-z-]+$")

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
			log := f.GetLog()

			// get all flags till "run"
			index := -1
			skipNext := false
			for i, v := range os.Args {
				if skipNext {
					skipNext = false
					continue
				} else if isFlag.MatchString(v) {
					skipNext = true
					continue
				}

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
			extraFlags, err := flagspkg.ApplyExtraFlags(cobraCmd, os.Args[:index])
			if err != nil {
				return err
			} else if len(extraFlags) > 0 {
				log.Infof("Applying extra flags from environment: %s", strings.Join(extraFlags, " "))
			}

			return cmd.RunRun(f, cobraCmd, os.Args[index:])
		},
	}

	return runCmd
}

// RunRun executes the functionality "devspace run"
func (cmd *RunCmd) RunRun(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configLoader := f.NewConfigLoader(cmd.ToConfigOptions(), f.GetLog())
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
