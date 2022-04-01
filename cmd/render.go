package cmd

import (
	"context"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	pipelinetypes "github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"io"
	"os"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// RenderCmd is a struct that defines a command call for "render"
type RenderCmd struct {
	*flags.GlobalFlags

	Tags []string

	SkipPush                bool
	SkipPushLocalKubernetes bool

	SkipBuild           bool
	ForceBuild          bool
	BuildSequential     bool
	MaxConcurrentBuilds int

	SkipDependencies bool
	SkipDependency   []string
	Dependency       []string

	Writer io.Writer
}

// NewRenderCmd creates a new devspace render command
func NewRenderCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &RenderCmd{
		GlobalFlags: globalFlags,
		Writer:      os.Stdout,
	}

	renderCmd := &cobra.Command{
		Use:   "render",
		Short: "Render builds all defined images and shows the yamls that would be deployed",
		Long: `
#######################################################
################## devspace render #####################
#######################################################
Builds all defined images and shows the yamls that would
be deployed via helm and kubectl, but skips actual 
deployment.
#######################################################`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f)
		},
	}

	renderCmd.Flags().BoolVarP(&cmd.ForceBuild, "force-build", "b", false, "Forces to build every image")
	renderCmd.Flags().BoolVar(&cmd.BuildSequential, "build-sequential", false, "Builds the images one after another instead of in parallel")
	renderCmd.Flags().IntVar(&cmd.MaxConcurrentBuilds, "max-concurrent-builds", 0, "The maximum number of image builds built in parallel (0 for infinite)")
	renderCmd.Flags().StringSliceVarP(&cmd.Tags, "tag", "t", []string{}, "Use the given tag for all built images")
	renderCmd.Flags().BoolVar(&cmd.SkipPush, "skip-push", false, "Skips image pushing, useful for minikube deployment")
	renderCmd.Flags().BoolVar(&cmd.SkipPushLocalKubernetes, "skip-push-local-kube", true, "Skips image pushing, if a local kubernetes environment is detected")
	renderCmd.Flags().BoolVar(&cmd.SkipBuild, "skip-build", false, "Skips image building")

	renderCmd.Flags().BoolVar(&cmd.SkipDependencies, "skip-dependencies", false, "Skips rendering the dependencies")
	renderCmd.Flags().StringSliceVar(&cmd.SkipDependency, "skip-dependency", []string{}, "Skips rendering the following dependencies")
	renderCmd.Flags().StringSliceVar(&cmd.Dependency, "dependency", []string{}, "Renders only the specific named dependencies")

	return renderCmd
}

// Run executes the command logic
func (cmd *RenderCmd) Run(f factory.Factory) error {
	f.GetLog().Warnf("This command is deprecated, please use 'devspace deploy --render' instead")
	configOptions := cmd.ToConfigOptions()
	ctx, err := prepare(context.Background(), f, configOptions, cmd.GlobalFlags, true)
	if err != nil {
		return err
	}

	return runWithHooks(ctx, "renderCommand", func() error {
		return cmd.runCommand(ctx, f, configOptions)
	})
}

func (cmd *RenderCmd) runCommand(ctx *devspacecontext.Context, f factory.Factory, configOptions *loader.ConfigOptions) error {
	return runPipeline(ctx, f, true, &PipelineOptions{
		Options: pipelinetypes.Options{
			BuildOptions: build.Options{
				Tags:                      cmd.Tags,
				SkipBuild:                 cmd.SkipBuild,
				SkipPush:                  cmd.SkipPush,
				SkipPushOnLocalKubernetes: cmd.SkipPushLocalKubernetes,
				ForceRebuild:              cmd.ForceBuild,
				Sequential:                cmd.BuildSequential,
				MaxConcurrentBuilds:       cmd.MaxConcurrentBuilds,
			},
			DeployOptions: deploy.Options{
				Render:       true,
				RenderWriter: cmd.Writer,
			},
			DependencyOptions: pipelinetypes.DependencyOptions{
				Exclude: cmd.SkipDependency,
				Only:    cmd.Dependency,
			},
		},
		ConfigOptions: configOptions,
		Pipeline:      "deploy",
	})
}
