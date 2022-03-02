package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/registry"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/dev"
	"github.com/loft-sh/devspace/pkg/devspace/devpod"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	fakekube "github.com/loft-sh/devspace/pkg/devspace/kubectl/testing"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/server"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/kubernetes/fake"
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
	VerboseDependencies bool
	Pipeline            string

	SkipPush                bool
	SkipPushLocalKubernetes bool
	Render                  bool
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
	deployCmd.Flags().BoolVar(&cmd.Render, "render", false, "If true will render manifests and print them instead of actually deploying them")
	deployCmd.Flags().StringVar(&cmd.Pipeline, "pipeline", "deploy", "The pipeline to execute")

	deployCmd.Flags().StringSliceVar(&cmd.SkipDependency, "skip-dependency", []string{}, "Skips deploying the following dependencies")
	deployCmd.Flags().StringSliceVar(&cmd.Dependency, "dependency", []string{}, "Deploys only the specific named dependencies")

	deployCmd.Flags().BoolVar(&cmd.Wait, "wait", false, "If true will wait for pods to be running or fails after given timeout")
	deployCmd.Flags().IntVar(&cmd.Timeout, "timeout", 120, "Timeout until deploy should stop waiting")

	return deployCmd
}

// Run executes the down command logic
func (cmd *DeployCmd) Run(f factory.Factory) error {
	configOptions := cmd.ToConfigOptions()
	ctx, err := prepare(f, configOptions, cmd.GlobalFlags, false)
	if err != nil {
		return err
	}

	return runWithHooks(ctx, "deployCommand", func() error {
		return cmd.runCommand(ctx, f, configOptions)
	})
}

func (cmd *DeployCmd) runCommand(ctx *devspacecontext.Context, f factory.Factory, configOptions *loader.ConfigOptions) error {
	return runPipeline(ctx, f, &PipelineOptions{
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
				Render:      cmd.Render,
				SkipDeploy:  cmd.SkipDeploy,
			},
			DependencyOptions: types.DependencyOptions{
				Exclude: cmd.SkipDependency,
			},
		},
		ConfigOptions: configOptions,
		Only:          cmd.Dependency,
		Pipeline:      cmd.Pipeline,
		Wait:          cmd.Wait,
		Timeout:       cmd.Timeout,
	})
}

func prepare(f factory.Factory, configOptions *loader.ConfigOptions, globalFlags *flags.GlobalFlags, allowFailingKubeClient bool) (*devspacecontext.Context, error) {
	log := f.GetLog()

	// set config root
	configLoader, err := f.NewConfigLoader(globalFlags.ConfigPath)
	if err != nil {
		return nil, err
	}
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return nil, err
	} else if !configExists {
		return nil, errors.New(message.ConfigNotFound)
	}

	// start file logging
	logpkg.StartFileLogging()

	// create kubectl client
	client, err := f.NewKubeClientFromContext(globalFlags.KubeContext, globalFlags.Namespace)
	if err != nil {
		if allowFailingKubeClient {
			log.Warnf("Unable to create new kubectl client: %v", err)
			log.Warn("Using fake client to render resources")
			log.WriteString(logrus.WarnLevel, "\n")

			kube := fake.NewSimpleClientset()
			client = &fakekube.Client{
				Client: kube,
			}
		} else {
			return nil, errors.Errorf("error creating Kubernetes client: %v. Please make sure you have a valid Kubernetes context that points to a working Kubernetes cluster. If in doubt, please check if the following command works locally: `kubectl get namespaces`", err)
		}
	}

	// load generated config
	localCache, err := configLoader.LoadLocalCache()
	if err != nil {
		return nil, errors.Errorf("error loading local cache: %v", err)
	}

	// If the current kube context or namespace is different than old,
	// show warnings and reset kube client if necessary
	client, err = client.CheckKubeContext(localCache, globalFlags.NoWarn, log)
	if err != nil {
		return nil, err
	}

	// Create our parent context
	backgroundCtx := context.Background()

	// load config
	configInterface, err := configLoader.LoadWithCache(backgroundCtx, localCache, client, configOptions, log)
	if err != nil {
		return nil, err
	}

	// create devspace context
	return devspacecontext.NewContext(backgroundCtx, log).
		WithConfig(configInterface).
		WithKubeClient(client), nil
}

type PipelineOptions struct {
	types.Options

	ConfigOptions *loader.ConfigOptions
	Only          []string
	Pipeline      string
	Wait          bool
	Timeout       int
	UIPort        int
}

func runPipeline(ctx *devspacecontext.Context, f factory.Factory, options *PipelineOptions) error {
	// create namespace if necessary
	if !options.DeployOptions.Render {
		err := ctx.KubeClient.EnsureNamespace(ctx.Context, ctx.KubeClient.Namespace(), ctx.Log)
		if err != nil {
			return errors.Errorf("unable to create namespace: %v", err)
		}
	}

	// create docker client
	dockerClient, err := f.NewDockerClient(ctx.Log)
	if err != nil {
		dockerClient = nil
	}

	// deploy dependencies
	dependencies, err := f.NewDependencyManager(ctx, options.ConfigOptions).ResolveAll(ctx, dependency.ResolveOptions{
		SkipDependencies: options.DependencyOptions.Exclude,
		Dependencies:     options.Only,
		Silent:           true,
		Verbose:          false,
	})
	if err != nil {
		return errors.Wrap(err, "deploy dependencies")
	}
	ctx = ctx.WithDependencies(dependencies)

	// start ui & open
	serv, err := startServices(ctx, options.UIPort)
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
	if ctx.Config.Config().Pipelines != nil && ctx.Config.Config().Pipelines[options.Pipeline] != nil {
		configPipeline = ctx.Config.Config().Pipelines[options.Pipeline]
	} else {
		configPipeline, err = pipeline.GetDefaultPipeline(options.Pipeline)
		if err != nil {
			return err
		}
	}

	// create dependency registry
	dependencyRegistry := registry.NewDependencyRegistry("http://"+serv.Server.Addr, options.DeployOptions.Render)

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
	pipe := pipeline.NewPipeline(options.Pipeline, devPodManager, dependencyRegistry, configPipeline, options.Options)

	// start pipeline
	err = pipe.Run(ctx.WithLogger(ctx.Log.WithoutPrefix()))
	if err != nil {
		return err
	}

	// wait for dev
	pipe.WaitDev()

	// wait if necessary
	if options.Wait {
		report, err := f.NewAnalyzer(ctx.KubeClient, f.GetLog()).CreateReport(ctx.KubeClient.Namespace(), analyze.Options{Wait: true, Patient: true, Timeout: options.Timeout, IgnorePodRestarts: true})
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
