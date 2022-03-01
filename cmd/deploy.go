package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/registry"
	"github.com/loft-sh/devspace/pkg/devspace/dev"
	"github.com/loft-sh/devspace/pkg/devspace/devpod"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/server"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"
	"gopkg.in/yaml.v3"
	"time"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/analyze"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/util/factory"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DeployCmd holds the required data for the down cmd
type DeployCmd struct {
	*flags.GlobalFlags

	ForceBuild          bool
	SkipBuild           bool
	BuildSequential     bool
	MaxConcurrentBuilds int

	ForceDeploy         bool
	SkipDeploy          bool
	Deployments         string
	VerboseDependencies bool
	Pipeline            string

	SkipPush                bool
	SkipPushLocalKubernetes bool
	Dependency              []string
	SkipDependency          []string

	Wait    bool
	Timeout int

	log logpkg.Logger
}

// NewDeployCmd creates a new deploy command
func NewDeployCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &DeployCmd{
		GlobalFlags: globalFlags,
		log:         f.GetLog(),
	}

	deployCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy the project",
		Long: `
#######################################################
################## devspace deploy ####################
#######################################################
Deploys the current project to a Space or namespace:

devspace deploy
devspace deploy -n some-namespace
devspace deploy --kube-context=deploy-context
#######################################################`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Print upgrade message if new version available
			upgrade.PrintUpgradeMessage()
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f)
		},
	}

	deployCmd.Flags().BoolVar(&cmd.VerboseDependencies, "verbose-dependencies", true, "Deploys the dependencies verbosely")

	deployCmd.Flags().BoolVar(&cmd.SkipPush, "skip-push", false, "Skips image pushing, useful for minikube deployment")
	deployCmd.Flags().BoolVar(&cmd.SkipPushLocalKubernetes, "skip-push-local-kube", true, "Skips image pushing, if a local kubernetes environment is detected")

	deployCmd.Flags().BoolVarP(&cmd.ForceBuild, "force-build", "b", false, "Forces to (re-)build every image")
	deployCmd.Flags().BoolVar(&cmd.SkipBuild, "skip-build", false, "Skips building of images")
	deployCmd.Flags().BoolVar(&cmd.BuildSequential, "build-sequential", false, "Builds the images one after another instead of in parallel")
	deployCmd.Flags().IntVar(&cmd.MaxConcurrentBuilds, "max-concurrent-builds", 0, "The maximum number of image builds built in parallel (0 for infinite)")

	deployCmd.Flags().BoolVarP(&cmd.ForceDeploy, "force-deploy", "d", false, "Forces to (re-)deploy every deployment")
	deployCmd.Flags().BoolVar(&cmd.SkipDeploy, "skip-deploy", false, "Skips deploying and only builds images")
	deployCmd.Flags().StringVar(&cmd.Deployments, "deployments", "", "Only deploy a specific deployment (You can specify multiple deployments comma-separated")
	deployCmd.Flags().StringVar(&cmd.Pipeline, "pipeline", "deploy", "The pipeline to execute")

	deployCmd.Flags().StringSliceVar(&cmd.SkipDependency, "skip-dependency", []string{}, "Skips deploying the following dependencies")
	deployCmd.Flags().StringSliceVar(&cmd.Dependency, "dependency", []string{}, "Deploys only the specific named dependencies")

	deployCmd.Flags().BoolVar(&cmd.Wait, "wait", false, "If true will wait for pods to be running or fails after given timeout")
	deployCmd.Flags().IntVar(&cmd.Timeout, "timeout", 120, "Timeout until deploy should stop waiting")

	return deployCmd
}

// Run executes the down command logic
func (cmd *DeployCmd) Run(f factory.Factory) error {
	// set config root
	cmd.log = f.GetLog()
	configOptions := cmd.ToConfigOptions()
	configLoader, err := f.NewConfigLoader(cmd.ConfigPath)
	if err != nil {
		return err
	}
	configExists, err := configLoader.SetDevSpaceRoot(cmd.log)
	if err != nil {
		return err
	} else if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// start file logging
	logpkg.StartFileLogging()

	// create kubectl client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace)
	if err != nil {
		return errors.Errorf("unable to create new kubectl client: %v", err)
	}

	// load generated config
	localCache, err := localcache.NewCacheLoaderFromDevSpacePath(cmd.ConfigPath).Load()
	if err != nil {
		return errors.Errorf("error loading generated.yaml: %v", err)
	}

	// If the current kube context or namespace is different than old,
	// show warnings and reset kube client if necessary
	client, err = client.CheckKubeContext(localCache, cmd.NoWarn, cmd.log)
	if err != nil {
		return err
	}

	// load config
	configInterface, err := configLoader.LoadWithCache(localCache, client, configOptions, cmd.log)
	if err != nil {
		return err
	}

	// create devspace context
	ctx := devspacecontext.NewContext(context.Background(), cmd.log).
		WithConfig(configInterface).
		WithKubeClient(client)

	return runWithHooks(ctx, "deployCommand", func() error {
		return cmd.runCommand(ctx, f, configOptions)
	})
}

func (cmd *DeployCmd) runCommand(ctx *devspacecontext.Context, f factory.Factory, configOptions *loader.ConfigOptions) error {
	err := runPipeline(ctx, f, configOptions, cmd.SkipDependency, cmd.Dependency, "deploy", `run_dependencies_pipeline --all
build_images --all
create_deployments --all`, cmd.Wait, cmd.Timeout, 0)
	if err != nil {
		return err
	}

	return nil
}

func runPipeline(
	ctx *devspacecontext.Context,
	f factory.Factory,
	configOptions *loader.ConfigOptions,
	exclude, only []string,
	executePipeline string,
	fallbackPipeline string,
	wait bool,
	timeout int,
	uiPort int,
) error {
	// create namespace if necessary
	err := ctx.KubeClient.EnsureNamespace(ctx.Context, ctx.KubeClient.Namespace(), ctx.Log)
	if err != nil {
		return errors.Errorf("unable to create namespace: %v", err)
	}

	// create docker client
	dockerClient, err := f.NewDockerClient(ctx.Log)
	if err != nil {
		dockerClient = nil
	}

	// deploy dependencies
	dependencies, err := f.NewDependencyManager(ctx, configOptions).ResolveAll(ctx, dependency.ResolveOptions{
		SkipDependencies: exclude,
		Dependencies:     only,
		Silent:           true,
		Verbose:          false,
	})
	if err != nil {
		return errors.Wrap(err, "deploy dependencies")
	}
	ctx = ctx.WithDependencies(dependencies)

	// start ui & open
	serv, err := startServices(ctx, uiPort)
	if err != nil {
		return err
	}

	// execute plugin hook
	err = hook.ExecuteHooks(ctx, nil, "deploy")
	if err != nil {
		return err
	}

	// create pull secrets if necessary
	err = f.NewPullSecretClient(dockerClient).EnsurePullSecrets(ctx, ctx.KubeClient.Namespace())
	if err != nil {
		ctx.Log.Warn(err)
	}

	// update last used kube context & save generated yaml
	err = updateLastKubeContext(ctx)
	if err != nil {
		return errors.Wrap(err, "update last kube context")
	}

	var configPipeline *latest.Pipeline
	if ctx.Config.Config().Pipelines != nil && ctx.Config.Config().Pipelines[executePipeline] != nil {
		configPipeline = ctx.Config.Config().Pipelines[executePipeline]
	} else {
		configPipeline = &latest.Pipeline{
			Steps: []latest.PipelineStep{
				{
					Run: fallbackPipeline,
				},
			},
		}
	}

	// create dependency registry
	dependencyRegistry := registry.NewDependencyRegistry("http://" + serv.Server.Addr)

	// exclude ourselves
	couldExclude, err := dependencyRegistry.MarkDependencyExcluded(ctx, ctx.Config.Config().Name, true)
	if err != nil {
		return err
	} else if !couldExclude {
		return fmt.Errorf("couldn't start project %s, because there is another DevSpace instance active in the current namespace right now that uses the same project", ctx.Config.Config().Name)
	}

	// create a new base dev pod manager
	devPodManager := devpod.NewManager(ctx.Context)
	defer devPodManager.Close()

	// marshal pipeline
	configPipelineBytes, err := yaml.Marshal(configPipeline)
	if err == nil {
		ctx.Log.Debugf("Run pipeline:\n%s\n", string(configPipelineBytes))
	}

	// get deploy pipeline
	pipe := pipeline.NewPipeline(executePipeline, devPodManager, dependencyRegistry, configPipeline)

	// start pipeline
	err = pipe.Run(ctx.WithLogger(ctx.Log.WithoutPrefix()))
	if err != nil {
		return err
	}

	// wait for dev
	pipe.WaitDev()

	// wait if necessary
	if wait {
		report, err := f.NewAnalyzer(ctx.KubeClient, f.GetLog()).CreateReport(ctx.KubeClient.Namespace(), analyze.Options{Wait: true, Patient: true, Timeout: timeout, IgnorePodRestarts: true})
		if err != nil {
			return errors.Wrap(err, "analyze")
		}

		if len(report) > 0 {
			return errors.Errorf(analyze.ReportToString(report))
		}
	}

	return nil
}

func startServices(ctx *devspacecontext.Context, uiPort int) (*server.Server, error) {
	// Open UI if configured
	serv, err := dev.UI(ctx, uiPort)
	if err != nil {
		return nil, err
	}

	// Run dev.open configs
	for _, openConfig := range ctx.Config.Config().Open {
		if openConfig.URL != "" {
			maxWait := 4 * time.Minute
			ctx.Log.Infof("Opening '%s' as soon as application will be started (timeout: %s)", openConfig.URL, maxWait)

			go func(url string) {
				// Use DiscardLogger as we do not want to print warnings about failed HTTP requests
				err := openURL(url, nil, "", logpkg.Discard, maxWait)
				if err != nil {
					// Use warn instead of fatal to prevent exit
					// Do not print warning
					// log.Warn(err)
					_ = err // just to avoid empty branch (SA9003) lint error
				}
			}(openConfig.URL)
		}
	}

	return serv, nil
}
