package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/context/values"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/registry"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/dev"
	"github.com/loft-sh/devspace/pkg/devspace/devpod"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	fakekube "github.com/loft-sh/devspace/pkg/devspace/kubectl/testing"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"
	"github.com/loft-sh/devspace/pkg/util/factory"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"k8s.io/client-go/kubernetes/fake"
	"os"
	"strings"
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

	log logpkg.Logger

	Ctx context.Context
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
	deployCmd.Flags().StringVar(&cmd.Pipeline, "pipeline", "", "The pipeline to execute")

	deployCmd.Flags().StringSliceVar(&cmd.SkipDependency, "skip-dependency", []string{}, "Skips deploying the following dependencies")
	deployCmd.Flags().StringSliceVar(&cmd.Dependency, "dependency", []string{}, "Deploys only the specific named dependencies")

	return deployCmd
}

// Run executes the down command logic
func (cmd *DeployCmd) Run(f factory.Factory) error {
	if cmd.Ctx == nil {
		var cancelFn context.CancelFunc
		cmd.Ctx, cancelFn = context.WithCancel(context.Background())
		defer cancelFn()
	}

	// set command in context
	cmd.Ctx = values.WithCommand(cmd.Ctx, "deploy")

	configOptions := cmd.ToConfigOptions()
	ctx, err := prepare(cmd.Ctx, f, configOptions, cmd.GlobalFlags, false)
	if err != nil {
		return err
	}

	return runWithHooks(ctx, "deployCommand", func() error {
		return cmd.runCommand(ctx, f, configOptions)
	})
}

func (cmd *DeployCmd) runCommand(ctx *devspacecontext.Context, f factory.Factory, configOptions *loader.ConfigOptions) error {
	if cmd.Pipeline == "" {
		cmd.Pipeline = "deploy"
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
				Render:      cmd.Render,
				SkipDeploy:  cmd.SkipDeploy,
			},
			DependencyOptions: types.DependencyOptions{
				Exclude: cmd.SkipDependency,
				Only:    cmd.Dependency,
			},
		},
		ConfigOptions: configOptions,
		Pipeline:      cmd.Pipeline,
	})
}

func prepare(ctx context.Context, f factory.Factory, configOptions *loader.ConfigOptions, globalFlags *flags.GlobalFlags, allowFailingKubeClient bool) (*devspacecontext.Context, error) {
	// start file logging
	logpkg.StartFileLogging()

	// create a temporary folder for us to use
	tempFolder, err := ioutil.TempDir("", "devspace-")
	if err != nil {
		return nil, errors.Wrap(err, "create temporary folder")
	}

	// add temp folder to context
	ctx = values.WithTempFolder(ctx, tempFolder)

	// get the main logger after file logging is started
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
	client, err = kubectl.CheckKubeContext(client, localCache, globalFlags.NoWarn, globalFlags.SwitchContext, log)
	if err != nil {
		return nil, err
	}

	// load config
	configInterface, err := configLoader.LoadWithCache(ctx, localCache, client, configOptions, log)
	if err != nil {
		return nil, err
	}

	// create devspace context
	return devspacecontext.NewContext(ctx, log).
		WithConfig(configInterface).
		WithKubeClient(client), nil
}

type PipelineOptions struct {
	types.Options

	ConfigOptions *loader.ConfigOptions
	Pipeline      string
	ShowUI        bool
	UIPort        int
}

func runPipeline(ctx *devspacecontext.Context, f factory.Factory, forceLeader bool, options *PipelineOptions) error {
	// create namespace if necessary
	if !options.DeployOptions.Render {
		err := kubectl.EnsureNamespace(ctx.Context, ctx.KubeClient, ctx.KubeClient.Namespace(), ctx.Log)
		if err != nil {
			return errors.Errorf("unable to create namespace: %v", err)
		}
	}

	// print config
	if ctx.Log.GetLevel() == logrus.DebugLevel {
		out, _ := yaml.Marshal(ctx.Config.Config())
		ctx.Log.Debugf("Use config:\n%s\n", string(out))
	}

	// resolve dependencies
	dependencies, err := f.NewDependencyManager(ctx, options.ConfigOptions).ResolveAll(ctx, dependency.ResolveOptions{})
	if err != nil {
		return errors.Wrap(err, "deploy dependencies")
	}
	ctx = ctx.WithDependencies(dependencies)

	// update last used kube context & save generated yaml
	err = updateLastKubeContext(ctx)
	if err != nil {
		return errors.Wrap(err, "update last kube context")
	}

	var configPipeline *latest.Pipeline
	if ctx.Config.Config().Pipelines != nil && ctx.Config.Config().Pipelines[options.Pipeline] != nil {
		configPipeline = ctx.Config.Config().Pipelines[options.Pipeline]
	} else {
		configPipeline, err = types.GetDefaultPipeline(options.Pipeline)
		if err != nil {
			return err
		}
	}

	// marshal pipeline
	configPipelineBytes, err := yaml.Marshal(configPipeline)
	if err == nil {
		ctx.Log.Debugf("Run pipeline:\n%s\n", string(configPipelineBytes))
	}

	// create a new base dev pod manager
	devPodManager := devpod.NewManager(ctx.Context)
	defer devPodManager.Close()

	// create dependency registry
	dependencyRegistry := registry.NewDependencyRegistry(options.DeployOptions.Render)

	// get deploy pipeline
	pipe := pipeline.NewPipeline(ctx.Config.Config().Name, devPodManager, dependencyRegistry, configPipeline, options.Options)

	// start ui & open
	serv, err := dev.UI(ctx, options.UIPort, options.ShowUI, pipe)
	if err != nil {
		return err
	}
	dependencyRegistry.SetServer("http://" + serv.Server.Addr)

	// exclude ourselves
	couldExclude, err := dependencyRegistry.MarkDependencyExcluded(ctx, ctx.Config.Config().Name, forceLeader)
	if err != nil {
		return err
	} else if !couldExclude {
		return fmt.Errorf("couldn't execute '%s', because there is another DevSpace instance active in the current namespace right now that uses the same project name (%s)", strings.Join(os.Args, " "), ctx.Config.Config().Name)
	}
	ctx.Log.Debugf("Marked project excluded: %v", ctx.Config.Config().Name)

	// get a stdout writer
	stdoutWriter := ctx.Log.Writer(ctx.Log.GetLevel(), true)
	defer stdoutWriter.Close()

	// get a stderr writer
	stderrWriter := ctx.Log.Writer(logrus.WarnLevel, true)
	defer stderrWriter.Close()

	// start pipeline
	err = pipe.Run(ctx.WithLogger(logpkg.NewStreamLoggerWithFormat(stdoutWriter, stderrWriter, ctx.Log.GetLevel(), logpkg.TimeFormat)))
	if err != nil {
		return err
	}
	ctx.Log.Debugf("Wait for dev to finish")

	// wait for dev
	err = pipe.WaitDev()
	if err != nil {
		return err
	}

	return nil
}
