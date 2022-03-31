package cmd

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/build"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/context/values"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// RunPipelineCmd holds the command flags
type RunPipelineCmd struct {
	*flags.GlobalFlags

	SkipPush                bool
	SkipPushLocalKubernetes bool

	Dependency     []string
	SkipDependency []string

	ForceBuild          bool
	SkipBuild           bool
	BuildSequential     bool
	MaxConcurrentBuilds int

	ForceDeploy bool
	SkipDeploy  bool

	DisableUI bool

	configLoader loader.ConfigLoader
	log          log.Logger

	// used for testing to allow interruption
	Ctx context.Context
}

// NewRunPipelineCmd creates a new devspace run-pipeline command
func NewRunPipelineCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &RunPipelineCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	runPipelineCmd := &cobra.Command{
		Use:   "run-pipeline",
		Short: "Starts the development mode",
		Long: `
#######################################################
############## devspace run-pipeline ##################
#######################################################
Execute a pipeline
#######################################################`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Print upgrade message if new version available
			upgrade.PrintUpgradeMessage()
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f, args)
		},
	}

	runPipelineCmd.Flags().StringSliceVar(&cmd.SkipDependency, "skip-dependency", []string{}, "Skips the following dependencies for deployment")
	runPipelineCmd.Flags().StringSliceVar(&cmd.Dependency, "dependency", []string{}, "Deploys only the specified named dependencies")

	runPipelineCmd.Flags().BoolVarP(&cmd.ForceBuild, "force-build", "b", false, "Forces to build every image")
	runPipelineCmd.Flags().BoolVar(&cmd.SkipBuild, "skip-build", false, "Skips building of images")
	runPipelineCmd.Flags().BoolVar(&cmd.BuildSequential, "build-sequential", false, "Builds the images one after another instead of in parallel")
	runPipelineCmd.Flags().IntVar(&cmd.MaxConcurrentBuilds, "max-concurrent-builds", 0, "The maximum number of image builds built in parallel (0 for infinite)")

	runPipelineCmd.Flags().BoolVarP(&cmd.ForceDeploy, "force-deploy", "d", false, "Forces to deploy every deployment")
	runPipelineCmd.Flags().BoolVar(&cmd.SkipDeploy, "skip-deploy", false, "If enabled will skip deploying")

	runPipelineCmd.Flags().BoolVar(&cmd.SkipPush, "skip-push", false, "Skips image pushing, useful for minikube deployment")
	runPipelineCmd.Flags().BoolVar(&cmd.SkipPushLocalKubernetes, "skip-push-local-kube", true, "Skips image pushing, if a local kubernetes environment is detected")

	runPipelineCmd.Flags().BoolVar(&cmd.DisableUI, "disable-ui", false, "Disables the ui server")
	return runPipelineCmd
}

// Run executes the command logic
func (cmd *RunPipelineCmd) Run(f factory.Factory, args []string) error {
	if cmd.Ctx == nil {
		var cancelFn context.CancelFunc
		cmd.Ctx, cancelFn = context.WithCancel(context.Background())
		defer cancelFn()
	}

	// set command in context
	cmd.Ctx = values.WithCommand(cmd.Ctx, "run-pipeline")
	configOptions := cmd.ToConfigOptions()
	ctx, err := prepare(cmd.Ctx, f, configOptions, cmd.GlobalFlags, false)
	if err != nil {
		return err
	}

	return runWithHooks(ctx, "devCommand", func() error {
		// Build and deploy images
		err = cmd.runCommand(ctx, f, configOptions, args[0])
		if err != nil {
			return err
		}

		return nil
	})
}

func (cmd *RunPipelineCmd) runCommand(ctx *devspacecontext.Context, f factory.Factory, configOptions *loader.ConfigOptions, pipeline string) error {
	return runPipeline(ctx, f, true, &PipelineOptions{
		Options: types.Options{
			BuildOptions: build.Options{
				SkipBuild:                 cmd.SkipBuild,
				SkipPush:                  cmd.SkipPush,
				SkipPushOnLocalKubernetes: cmd.SkipPushLocalKubernetes,
				ForceRebuild:              cmd.ForceBuild,
				Sequential:                cmd.BuildSequential,
				MaxConcurrentBuilds:       cmd.MaxConcurrentBuilds,
			},
			DeployOptions: deploy.Options{
				ForceDeploy: cmd.ForceDeploy,
				SkipDeploy:  cmd.SkipDeploy,
			},
			DependencyOptions: types.DependencyOptions{
				Exclude: cmd.SkipDependency,
				Only:    cmd.Dependency,
			},
		},
		ConfigOptions: configOptions,
		Pipeline:      pipeline,
		ShowUI:        !cmd.DisableUI,
	})
}
