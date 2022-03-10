package cmd

import (
	"context"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/context/values"
	pipelinetypes "github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// BuildCmd is a struct that defines a command call for "build"
type BuildCmd struct {
	*flags.GlobalFlags

	Tags []string

	SkipPush                bool
	SkipPushLocalKubernetes bool
	SkipDependency          []string
	Dependency              []string

	Pipeline string

	ForceBuild          bool
	BuildSequential     bool
	MaxConcurrentBuilds int

	Ctx context.Context
}

// NewBuildCmd creates a new devspace build command
func NewBuildCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &BuildCmd{GlobalFlags: globalFlags}

	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Builds all defined images and pushes them",
		Long: `
#######################################################
################## devspace build #####################
#######################################################
Builds all defined images and pushes them
#######################################################`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f)
		},
	}

	buildCmd.Flags().BoolVarP(&cmd.ForceBuild, "force-build", "b", false, "Forces to build every image")
	buildCmd.Flags().BoolVar(&cmd.BuildSequential, "build-sequential", false, "Builds the images one after another instead of in parallel")
	buildCmd.Flags().IntVar(&cmd.MaxConcurrentBuilds, "max-concurrent-builds", 0, "The maximum number of image builds built in parallel (0 for infinite)")

	buildCmd.Flags().StringSliceVarP(&cmd.Tags, "tag", "t", []string{}, "Use the given tag for all built images")
	buildCmd.Flags().StringSliceVar(&cmd.SkipDependency, "skip-dependency", []string{}, "Skips building the following dependencies")
	buildCmd.Flags().StringSliceVar(&cmd.Dependency, "dependency", []string{}, "Builds only the specific named dependencies")
	buildCmd.Flags().StringVar(&cmd.Pipeline, "pipeline", "", "The pipeline to execute")

	buildCmd.Flags().BoolVar(&cmd.SkipPush, "skip-push", false, "Skips image pushing, useful for minikube deployment")
	buildCmd.Flags().BoolVar(&cmd.SkipPushLocalKubernetes, "skip-push-local-kube", false, "Skips image pushing, if a local kubernetes environment is detected")

	return buildCmd
}

// Run executes the command logic
func (cmd *BuildCmd) Run(f factory.Factory) error {
	if cmd.Ctx == nil {
		var cancelFn context.CancelFunc
		cmd.Ctx, cancelFn = context.WithCancel(context.Background())
		defer cancelFn()
	}

	// set command in context
	cmd.Ctx = values.WithCommand(cmd.Ctx, "build")

	configOptions := cmd.ToConfigOptions()
	ctx, err := prepare(cmd.Ctx, f, configOptions, cmd.GlobalFlags, true)
	if err != nil {
		return err
	}

	return runWithHooks(ctx, "buildCommand", func() error {
		return cmd.runCommand(ctx, f, configOptions)
	})
}

func (cmd *BuildCmd) runCommand(ctx *devspacecontext.Context, f factory.Factory, configOptions *loader.ConfigOptions) error {
	if cmd.Pipeline == "" {
		cmd.Pipeline = "build"
	}

	return runPipeline(ctx, f, true, &PipelineOptions{
		Options: pipelinetypes.Options{
			BuildOptions: build.Options{
				Tags:                      cmd.Tags,
				SkipPush:                  cmd.SkipPush,
				SkipPushOnLocalKubernetes: cmd.SkipPushLocalKubernetes,
				ForceRebuild:              cmd.ForceBuild,
				Sequential:                cmd.BuildSequential,
				MaxConcurrentBuilds:       cmd.MaxConcurrentBuilds,
			},
			DependencyOptions: pipelinetypes.DependencyOptions{
				Exclude: cmd.SkipDependency,
			},
		},
		ConfigOptions: configOptions,
		Only:          cmd.Dependency,
		Pipeline:      cmd.Pipeline,
	})
}
