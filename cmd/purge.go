package cmd

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/context/values"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"

	"github.com/loft-sh/devspace/pkg/util/factory"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// PurgeCmd holds the required data for the purge cmd
type PurgeCmd struct {
	*flags.GlobalFlags

	Deployments string
	Pipeline    string
	All         bool

	SkipDependency []string
	Dependency     []string

	log log.Logger

	Ctx context.Context
}

// NewPurgeCmd creates a new purge command
func NewPurgeCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &PurgeCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	purgeCmd := &cobra.Command{
		Use:   "purge",
		Short: "Delete deployed resources",
		Long: `
#######################################################
################### devspace purge ####################
#######################################################
Deletes the deployed kuberenetes resources:

devspace purge
devspace purge --dependencies
devspace purge -d my-deployment
#######################################################`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f)
		},
	}

	purgeCmd.Flags().StringVarP(&cmd.Deployments, "deployments", "d", "", "The deployment to delete (You can specify multiple deployments comma-separated, e.g. devspace-default,devspace-database etc.)")
	purgeCmd.Flags().BoolVarP(&cmd.All, "all", "a", true, "When enabled purges the dependencies as well")
	purgeCmd.Flags().StringVar(&cmd.Pipeline, "pipeline", "", "The pipeline to execute")

	purgeCmd.Flags().StringSliceVar(&cmd.SkipDependency, "skip-dependency", []string{}, "Skips the following dependencies from purging")
	purgeCmd.Flags().StringSliceVar(&cmd.Dependency, "dependency", []string{}, "Purges only the specific named dependencies")
	return purgeCmd
}

// Run executes the purge command logic
func (cmd *PurgeCmd) Run(f factory.Factory) error {
	if cmd.Ctx == nil {
		var cancelFn context.CancelFunc
		cmd.Ctx, cancelFn = context.WithCancel(context.Background())
		defer cancelFn()
	}

	// set command in context
	cmd.Ctx = values.WithCommand(cmd.Ctx, "purge")

	configOptions := cmd.ToConfigOptions()
	ctx, err := prepare(cmd.Ctx, f, configOptions, cmd.GlobalFlags, false)
	if err != nil {
		return err
	}

	return runWithHooks(ctx, "purgeCommand", func() error {
		return cmd.runCommand(ctx, f, configOptions)
	})
}

func (cmd *PurgeCmd) runCommand(ctx *devspacecontext.Context, f factory.Factory, configOptions *loader.ConfigOptions) error {
	if cmd.Pipeline == "" {
		cmd.Pipeline = "purge"
	}

	return runPipeline(ctx, f, false, &PipelineOptions{
		Options: types.Options{
			DependencyOptions: types.DependencyOptions{
				Exclude: cmd.SkipDependency,
				Only:    cmd.Dependency,
			},
		},
		ConfigOptions: configOptions,
		Pipeline:      cmd.Pipeline,
	})
}
