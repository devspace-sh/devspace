package cmd

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/util/interrupt"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"io"
	"os"

	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DevCmd is a struct that defines a command call for "up"
type DevCmd struct {
	*flags.GlobalFlags

	SkipPush                bool
	SkipPushLocalKubernetes bool
	VerboseDependencies     bool
	Open                    bool

	Dependency     []string
	SkipDependency []string

	ForceBuild          bool
	SkipBuild           bool
	BuildSequential     bool
	MaxConcurrentBuilds int

	ForceDeploy       bool
	Deployments       string
	ForceDependencies bool

	Sync            bool
	ExitAfterDeploy bool
	SkipPipeline    bool
	Portforwarding  bool
	VerboseSync     bool
	PrintSyncLog    bool

	UI     bool
	UIPort int

	Terminal          bool
	TerminalReconnect bool
	WorkingDirectory  string
	Interactive       bool

	Wait    bool
	Timeout int

	configLoader loader.ConfigLoader
	log          log.Logger

	// used for testing to allow interruption
	Interrupt chan error
	Stdout    io.Writer
	Stderr    io.Writer
	Stdin     io.Reader
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
Starts your project in development mode:
1. Builds your Docker images and override entrypoints if specified
2. Deploys the deployments via helm or kubectl
3. Forwards container ports to the local computer
4. Starts the sync client
5. Streams the logs of deployed containers

Open terminal instead of logs:
- Use "devspace dev -t" for opening a terminal
#######################################################`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Print upgrade message if new version available
			upgrade.PrintUpgradeMessage()
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f, args)
		},
	}

	devCmd.Flags().StringSliceVar(&cmd.SkipDependency, "skip-dependency", []string{}, "Skips the following dependencies for deployment")
	devCmd.Flags().StringSliceVar(&cmd.Dependency, "dependency", []string{}, "Deploys only the specified named dependencies")
	devCmd.Flags().BoolVar(&cmd.VerboseDependencies, "verbose-dependencies", true, "Deploys the dependencies verbosely")
	devCmd.Flags().BoolVar(&cmd.ForceDependencies, "force-dependencies", true, "Forces to re-evaluate dependencies (use with --force-build --force-deploy to actually force building & deployment of dependencies)")

	devCmd.Flags().BoolVarP(&cmd.ForceBuild, "force-build", "b", false, "Forces to build every image")
	devCmd.Flags().BoolVar(&cmd.SkipBuild, "skip-build", false, "Skips building of images")
	devCmd.Flags().BoolVar(&cmd.BuildSequential, "build-sequential", false, "Builds the images one after another instead of in parallel")
	devCmd.Flags().IntVar(&cmd.MaxConcurrentBuilds, "max-concurrent-builds", 0, "The maximum number of image builds built in parallel (0 for infinite)")

	devCmd.Flags().BoolVarP(&cmd.ForceDeploy, "force-deploy", "d", false, "Forces to deploy every deployment")
	devCmd.Flags().StringVar(&cmd.Deployments, "deployments", "", "Only deploy a specific deployment (You can specify multiple deployments comma-separated")

	devCmd.Flags().BoolVarP(&cmd.SkipPipeline, "skip-pipeline", "x", false, "Skips build & deployment and only starts sync, portforwarding & terminal")
	devCmd.Flags().BoolVar(&cmd.SkipPush, "skip-push", false, "Skips image pushing, useful for minikube deployment")
	devCmd.Flags().BoolVar(&cmd.SkipPushLocalKubernetes, "skip-push-local-kube", true, "Skips image pushing, if a local kubernetes environment is detected")

	devCmd.Flags().BoolVar(&cmd.UI, "ui", true, "Start the ui server")
	devCmd.Flags().IntVar(&cmd.UIPort, "ui-port", 0, "The port to use when opening the ui server")
	devCmd.Flags().BoolVar(&cmd.Open, "open", true, "Open defined URLs in the browser, if defined")
	devCmd.Flags().BoolVar(&cmd.Sync, "sync", true, "Enable code synchronization")
	devCmd.Flags().BoolVar(&cmd.VerboseSync, "verbose-sync", false, "When enabled the sync will log every file change")
	devCmd.Flags().BoolVar(&cmd.PrintSyncLog, "print-sync", false, "If enabled will print the sync log to the terminal")

	devCmd.Flags().BoolVar(&cmd.Portforwarding, "portforwarding", true, "Enable port forwarding")

	devCmd.Flags().BoolVar(&cmd.ExitAfterDeploy, "exit-after-deploy", false, "Exits the command after building the images and deploying the project")
	devCmd.Flags().BoolVarP(&cmd.Terminal, "terminal", "t", false, "Open a terminal instead of showing logs")
	devCmd.Flags().BoolVar(&cmd.TerminalReconnect, "terminal-reconnect", true, "Will try to reconnect the terminal if an unexpected exit code was encountered")
	devCmd.Flags().StringVar(&cmd.WorkingDirectory, "workdir", "", "The working directory where to open the terminal or execute the command")

	devCmd.Flags().BoolVar(&cmd.Wait, "wait", false, "If true will wait first for pods to be running or fails after given timeout")
	devCmd.Flags().IntVar(&cmd.Timeout, "timeout", 120, "Timeout until dev should stop waiting and fail")

	return devCmd
}

// Run executes the command logic
func (cmd *DevCmd) Run(f factory.Factory, args []string) error {
	if cmd.Interactive {
		cmd.log.Warn("Interactive mode flag is deprecated and will be removed in the future. Please take a look at https://devspace.sh/cli/docs/guides/interactive-mode on how to transition to an interactive profile")
	}

	// Set config root
	cmd.log = f.GetLog()
	var err error
	cmd.configLoader, err = f.NewConfigLoader(cmd.ConfigPath)
	if err != nil {
		return err
	}
	configOptions := cmd.ToConfigOptions()
	configExists, err := cmd.configLoader.SetDevSpaceRoot(cmd.log)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Start file logging
	log.StartFileLogging()

	// Create kubectl client and switch context if specified
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	// load local cache
	localCache, err := localcache.NewCacheLoaderFromDevSpacePath(cmd.ConfigPath).Load()
	if err != nil {
		return err
	}

	// If the current kube context or namespace is different than old,
	// show warnings and reset kube client if necessary
	client, err = client.CheckKubeContext(localCache, cmd.NoWarn, cmd.log)
	if err != nil {
		return err
	}

	// Load config
	configInterface, err := cmd.configLoader.LoadWithCache(localCache, client, configOptions, cmd.log)
	if err != nil {
		return err
	}

	// Get the config
	err = cmd.adjustConfig(configInterface)
	if err != nil {
		return err
	}

	// Create the devspace context
	ctx := devspacecontext.NewContext(context.Background(), cmd.log).
		WithConfig(configInterface).
		WithKubeClient(client)

	// Create namespace if necessary
	err = client.EnsureNamespace(ctx.Context, ctx.KubeClient.Namespace(), cmd.log)
	if err != nil {
		return errors.Errorf("Unable to create namespace: %v", err)
	}

	return runWithHooks(ctx, "devCommand", func() error {
		// Execute plugin hook
		err = hook.ExecuteHooks(ctx, nil, "dev")
		if err != nil {
			return err
		}

		// Build and deploy images
		err = cmd.runCommand(ctx, f, configOptions)
		if err != nil {
			return err
		}

		return nil
	})
}

func (cmd *DevCmd) runCommand(ctx *devspacecontext.Context, f factory.Factory, configOptions *loader.ConfigOptions) error {
	err := runPipeline(ctx, f, configOptions, cmd.SkipDependency, cmd.Dependency, "dev", `run_dependencies --all
build --all
deploy --all
dev --all`, cmd.Wait, cmd.Timeout, 0)
	if err != nil {
		return err
	}

	return nil
}

func runWithHooks(ctx *devspacecontext.Context, command string, fn func() error) (err error) {
	err = hook.ExecuteHooks(ctx, nil, command+":before:execute")
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			hook.LogExecuteHooks(ctx, map[string]interface{}{"error": err}, command+":after:execute", command+":error")
		} else {
			err = hook.ExecuteHooks(ctx, nil, command+":after:execute")
		}
	}()

	return interrupt.Global.Run(fn, func() {
		hook.LogExecuteHooks(ctx, nil, command+":interrupt")
	})
}

func (cmd *DevCmd) adjustConfig(conf config.Config) error {
	// check if terminal is enabled
	c := conf.Config()
	if cmd.Terminal {
		if len(c.Dev) == 0 {
			return errors.New("No image available in devspace config")
		}

		imageNames := make([]string, 0, len(c.Dev))
		for k, v := range c.Dev {
			v.Terminal = nil
			imageNames = append(imageNames, k)
		}

		// if only one image exists, use it, otherwise show image picker
		imageName := ""
		if len(imageNames) == 1 {
			imageName = imageNames[0]
		} else {
			var err error
			imageName, err = cmd.log.Question(&survey.QuestionOptions{
				Question: "Where do you want to open a terminal to?",
				Options:  imageNames,
			})
			if err != nil {
				return err
			}
		}
		c.Dev[imageName].Terminal = &latest.Terminal{}
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
