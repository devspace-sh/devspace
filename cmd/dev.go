package cmd

import (
	"context"
	"io"
	"os"

	"github.com/loft-sh/devspace/pkg/devspace/context/values"

	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/util/interrupt"
	"github.com/loft-sh/devspace/pkg/util/survey"

	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DevCmd holds the command options
type DevCmd struct {
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

	UIPort int

	Terminal         bool
	WorkingDirectory string
	Pipeline         string

	ShowUI bool

	configLoader loader.ConfigLoader
	log          log.Logger

	// used for testing to allow interruption
	Ctx context.Context
}

// NewDevCmd creates a new devspace dev command
func NewDevCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &DevCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	devCmd := &cobra.Command{
		Use:   "dev",
		Short: "Starts the development mode",
		Long: `
#######################################################
################### devspace dev ######################
#######################################################
Starts your project in development mode
#######################################################`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Print upgrade message if new version available
			upgrade.PrintUpgradeMessage()
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f)
		},
	}

	devCmd.Flags().StringSliceVar(&cmd.SkipDependency, "skip-dependency", []string{}, "Skips the following dependencies for deployment")
	devCmd.Flags().StringSliceVar(&cmd.Dependency, "dependency", []string{}, "Deploys only the specified named dependencies")

	devCmd.Flags().BoolVarP(&cmd.ForceBuild, "force-build", "b", false, "Forces to build every image")
	devCmd.Flags().BoolVar(&cmd.SkipBuild, "skip-build", false, "Skips building of images")
	devCmd.Flags().BoolVar(&cmd.BuildSequential, "build-sequential", false, "Builds the images one after another instead of in parallel")
	devCmd.Flags().IntVar(&cmd.MaxConcurrentBuilds, "max-concurrent-builds", 0, "The maximum number of image builds built in parallel (0 for infinite)")

	devCmd.Flags().BoolVarP(&cmd.ForceDeploy, "force-deploy", "d", false, "Forces to deploy every deployment")
	devCmd.Flags().BoolVar(&cmd.SkipDeploy, "skip-deploy", false, "If enabled will skip deploying")

	devCmd.Flags().BoolVar(&cmd.SkipPush, "skip-push", false, "Skips image pushing, useful for minikube deployment")
	devCmd.Flags().BoolVar(&cmd.SkipPushLocalKubernetes, "skip-push-local-kube", true, "Skips image pushing, if a local kubernetes environment is detected")

	devCmd.Flags().BoolVar(&cmd.ShowUI, "show-ui", false, "Shows the ui server")

	devCmd.Flags().IntVar(&cmd.UIPort, "ui-port", 0, "The port to use when opening the ui server")
	devCmd.Flags().StringVar(&cmd.Pipeline, "pipeline", "", "The pipeline to execute")

	devCmd.Flags().BoolVarP(&cmd.Terminal, "terminal", "t", false, "Open a terminal instead of showing logs")
	devCmd.Flags().StringVar(&cmd.WorkingDirectory, "workdir", "", "The working directory where to open the terminal or execute the command")

	return devCmd
}

// Run executes the command logic
func (cmd *DevCmd) Run(f factory.Factory) error {
	if cmd.Ctx == nil {
		var cancelFn context.CancelFunc
		cmd.Ctx, cancelFn = context.WithCancel(context.Background())
		defer cancelFn()
	}

	// set command in context
	cmd.Ctx = values.WithCommand(cmd.Ctx, "dev")

	configOptions := cmd.ToConfigOptions()
	ctx, err := prepare(cmd.Ctx, f, configOptions, cmd.GlobalFlags, false)
	if err != nil {
		return err
	}

	// Adjust the config
	err = cmd.adjustConfig(ctx.Config)
	if err != nil {
		return err
	}

	return runWithHooks(ctx, "devCommand", func() error {
		// Build and deploy images
		err = cmd.runCommand(ctx, f, configOptions)
		if err != nil {
			return err
		}

		return nil
	})
}

func (cmd *DevCmd) runCommand(ctx *devspacecontext.Context, f factory.Factory, configOptions *loader.ConfigOptions) error {
	if cmd.Pipeline == "" {
		cmd.Pipeline = "dev"
	}

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
		Pipeline:      cmd.Pipeline,
		ShowUI:        cmd.ShowUI,
		UIPort:        cmd.UIPort,
	})
}

func runWithHooks(ctx *devspacecontext.Context, command string, fn func() error) (err error) {
	err = hook.ExecuteHooks(ctx, nil, command+":before:execute")
	if err != nil {
		return err
	}

	defer func() {
		// delete temp folder
		deleteTempFolder(ctx.Context, ctx.Log)

		// execute hooks
		if err != nil {
			hook.LogExecuteHooks(ctx, map[string]interface{}{"error": err}, command+":after:execute", command+":error")
		} else {
			err = hook.ExecuteHooks(ctx, nil, command+":after:execute")
		}
	}()

	return interrupt.Global.Run(fn, func() {
		// delete temp folder
		deleteTempFolder(ctx.Context, ctx.Log)

		// execute hooks
		hook.LogExecuteHooks(ctx, nil, command+":interrupt")
	})
}

func deleteTempFolder(ctx context.Context, log log.Logger) {
	// delete temp folder
	tempFolder, ok := values.TempFolderFrom(ctx)
	if ok && tempFolder != os.TempDir() {
		err := os.RemoveAll(tempFolder)
		if err != nil {
			log.Debugf("error removing temp folder: %v", err)
		}
	}
}

func (cmd *DevCmd) adjustConfig(conf config.Config) error {
	// check if terminal is enabled
	c := conf.Config()
	if cmd.Terminal {
		if len(c.Dev) == 0 {
			return errors.New("No dev available in DevSpace config")
		}

		devNames := make([]string, 0, len(c.Dev))
		for k, v := range c.Dev {
			v.Terminal = nil
			devNames = append(devNames, k)
		}

		// if only one image exists, use it, otherwise show image picker
		devName := ""
		if len(devNames) == 1 {
			devName = devNames[0]
		} else {
			var err error
			devName, err = cmd.log.Question(&survey.QuestionOptions{
				Question: "Where do you want to open a terminal to?",
				Options:  devNames,
			})
			if err != nil {
				return err
			}
		}
		c.Dev[devName].Terminal = &latest.Terminal{
			WorkDir: cmd.WorkingDirectory,
		}
	}

	return nil
}

func defaultStdStreams(stdout io.Writer, stderr io.Writer, stdin io.Reader) (io.Writer, io.Writer, io.Reader) {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	if stdin == nil {
		stdin = os.Stdin
	}
	return stdout, stderr, stdin
}

func updateLastKubeContext(ctx *devspacecontext.Context) error {
	// Update generated if we deploy the application
	if ctx.Config != nil && ctx.Config.LocalCache() != nil {
		ctx.Config.LocalCache().SetLastContext(&localcache.LastContextConfig{
			Context:   ctx.KubeClient.CurrentContext(),
			Namespace: ctx.KubeClient.Namespace(),
		})

		err := ctx.Config.LocalCache().Save()
		if err != nil {
			return errors.Wrap(err, "save generated")
		}
	}

	return nil
}
