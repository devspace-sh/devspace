package cmd

import (
	"context"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
	"io"
	"os"
)

// OverwriteCmd holds the cmd flags of an overwrite command
type OverwriteCmd struct {
	*flags.GlobalFlags

	Command   *latest.CommandConfig
	Variables map[string]interface{}

	Stdout io.Writer
	Stderr io.Writer
}

// NewOverwriteCmd creates a new overwrite command
func NewOverwriteCmd(f factory.Factory, globalFlags *flags.GlobalFlags, command *latest.CommandConfig, variables map[string]interface{}) *cobra.Command {
	cmd := &OverwriteCmd{
		GlobalFlags: globalFlags,
		Command:     command,
		Variables:   variables,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
	}

	description := command.Description
	longDescription := command.Description
	if description == "" {
		description = "Runs command: " + command.Name
		longDescription = description
	}
	if len(description) > 64 {
		if len(description) > 64 {
			description = description[:61] + "..."
		}
	}

	runCmd := &cobra.Command{
		Use:                command.Name,
		DisableFlagParsing: true,
		Short:              description,
		Long:               longDescription,
		Args:               cobra.ArbitraryArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			args, err := ParseArgs(cobraCmd, cmd.GlobalFlags, f.GetLog())
			if err != nil {
				return err
			}

			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f, args)
		},
	}
	return runCmd
}

func (cmd *OverwriteCmd) Run(f factory.Factory, args []string) error {
	devCtx := devspacecontext.NewContext(context.Background(), f.GetLog())
	return ExecuteCommand(devCtx.Context, cmd.Command, cmd.Variables, args, devCtx.WorkingDir, cmd.Stdout, cmd.Stderr, os.Stdin)
}
