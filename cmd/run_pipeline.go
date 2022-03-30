package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/context/values"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/registry"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/dev"
	"github.com/loft-sh/devspace/pkg/devspace/devpod"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	fakekube "github.com/loft-sh/devspace/pkg/devspace/kubectl/testing"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/interrupt"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"k8s.io/client-go/kubernetes/fake"
	"os"
	"strings"
)

// RunPipelineCmd holds the command flags
type RunPipelineCmd struct {
	*flags.GlobalFlags

	Tags                    []string
	Render                  bool
	Pipeline                string
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

	Terminal bool

	ShowUI bool

	// used for testing to allow interruption
	Ctx          context.Context
	RenderWriter io.Writer

	configLoader loader.ConfigLoader
	log          log.Logger
}

func (cmd *RunPipelineCmd) AddFlags(command *cobra.Command, defaultPipeline string) {
	command.Flags().StringSliceVar(&cmd.SkipDependency, "skip-dependency", []string{}, "Skips the following dependencies for deployment")
	command.Flags().StringSliceVar(&cmd.Dependency, "dependency", []string{}, "Deploys only the specified named dependencies")

	command.Flags().BoolVarP(&cmd.ForceBuild, "force-build", "b", false, "Forces to build every image")
	command.Flags().BoolVar(&cmd.SkipBuild, "skip-build", false, "Skips building of images")
	command.Flags().BoolVar(&cmd.BuildSequential, "build-sequential", false, "Builds the images one after another instead of in parallel")
	command.Flags().IntVar(&cmd.MaxConcurrentBuilds, "max-concurrent-builds", 0, "The maximum number of image builds built in parallel (0 for infinite)")
	command.Flags().BoolVar(&cmd.Render, "render", false, "If true will render manifests and print them instead of actually deploying them")

	command.Flags().BoolVarP(&cmd.ForceDeploy, "force-deploy", "d", false, "Forces to deploy every deployment")
	command.Flags().BoolVar(&cmd.SkipDeploy, "skip-deploy", false, "If enabled will skip deploying")
	command.Flags().StringVar(&cmd.Pipeline, "pipeline", defaultPipeline, "The pipeline to execute")

	command.Flags().StringSliceVarP(&cmd.Tags, "tag", "t", []string{}, "Use the given tag for all built images")
	command.Flags().BoolVar(&cmd.SkipPush, "skip-push", false, "Skips image pushing, useful for minikube deployment")
	command.Flags().BoolVar(&cmd.SkipPushLocalKubernetes, "skip-push-local-kube", true, "Skips image pushing, if a local kubernetes environment is detected")

	command.Flags().BoolVarP(&cmd.Terminal, "terminal", "t", false, "Open a terminal instead of showing logs")
	command.Flags().BoolVar(&cmd.ShowUI, "show-ui", false, "Shows the ui server")
}

// NewRunPipelineCmd creates a new devspace run-pipeline command
func NewRunPipelineCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &RunPipelineCmd{GlobalFlags: globalFlags}
	runPipelineCmd := &cobra.Command{
		Use:   "run-pipeline",
		Short: "Starts the development mode",
		Long: `
#######################################################
############## devspace run-pipeline ##################
#######################################################
Execute a pipeline
#######################################################`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Pipeline == "" {
				return fmt.Errorf("please specify a pipeline through --pipeline or argument")
			} else if len(args) == 1 && cmd.Pipeline != "" {
				return fmt.Errorf("please specify a pipeline either through --pipeline or argument")
			} else if len(args) == 1 {
				cmd.Pipeline = args[0]
			}

			return cmd.Run(cobraCmd, args, f, "run-pipeline", "runPipelineCommand")
		},
	}

	cmd.AddFlags(runPipelineCmd, "")
	return runPipelineCmd
}

func (cmd *RunPipelineCmd) RunDefault(f factory.Factory) error {
	return cmd.Run(nil, nil, f, "run-pipeline", "runPipelineCommand")
}

// Run executes the command logic
func (cmd *RunPipelineCmd) Run(cobraCmd *cobra.Command, args []string, f factory.Factory, commandName, hookName string) error {
	if cmd.log == nil {
		cmd.log = f.GetLog()
	}
	if cmd.Silent {
		cmd.log.SetLevel(logrus.FatalLevel)
	}

	// Print upgrade message if new version available
	if !cmd.Render {
		upgrade.PrintUpgradeMessage(cmd.log)
	}
	if cobraCmd != nil {
		plugin.SetPluginCommand(cobraCmd, args)
	}

	if cmd.Ctx == nil {
		var cancelFn context.CancelFunc
		cmd.Ctx, cancelFn = context.WithCancel(context.Background())
		defer cancelFn()
	}

	// set command in context
	cmd.Ctx = values.WithCommand(cmd.Ctx, commandName)
	configOptions := cmd.ToConfigOptions()
	ctx, err := cmd.prepare(cmd.Ctx, f, configOptions, cmd.GlobalFlags, false)
	if err != nil {
		return err
	}

	return runWithHooks(ctx, hookName, func() error {
		return cmd.runPipeline(ctx, f, configOptions)
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

func (cmd *RunPipelineCmd) BuildOptions(configOptions *loader.ConfigOptions) *PipelineOptions {
	return &PipelineOptions{
		Options: types.Options{
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
				ForceDeploy:  cmd.ForceDeploy,
				Render:       cmd.Render,
				RenderWriter: cmd.RenderWriter,
				SkipDeploy:   cmd.SkipDeploy,
			},
			DependencyOptions: types.DependencyOptions{
				Exclude: cmd.SkipDependency,
				Only:    cmd.Dependency,
			},
		},
		ConfigOptions: configOptions,
		Pipeline:      cmd.Pipeline,
		ShowUI:        cmd.ShowUI,
	}
}

func (cmd *RunPipelineCmd) runPipeline(ctx *devspacecontext.Context, f factory.Factory, configOptions *loader.ConfigOptions) error {
	return runPipeline(ctx, f, true, cmd.BuildOptions(configOptions))
}

func (cmd *RunPipelineCmd) prepare(ctx context.Context, f factory.Factory, configOptions *loader.ConfigOptions, globalFlags *flags.GlobalFlags, allowFailingKubeClient bool) (*devspacecontext.Context, error) {
	// start file logging
	log.StartFileLogging()

	// create a temporary folder for us to use
	tempFolder, err := ioutil.TempDir("", "devspace-")
	if err != nil {
		return nil, errors.Wrap(err, "create temporary folder")
	}

	// add temp folder to context
	ctx = values.WithTempFolder(ctx, tempFolder)

	// set config root
	configLoader, err := f.NewConfigLoader(globalFlags.ConfigPath)
	if err != nil {
		return nil, err
	}
	configExists, err := configLoader.SetDevSpaceRoot(cmd.log)
	if err != nil {
		return nil, err
	} else if !configExists {
		return nil, errors.New(message.ConfigNotFound)
	}

	// create kubectl client
	client, err := f.NewKubeClientFromContext(globalFlags.KubeContext, globalFlags.Namespace)
	if err != nil {
		if allowFailingKubeClient {
			cmd.log.Warnf("Unable to create new kubectl client: %v", err)
			cmd.log.Warn("Using fake client to render resources")
			cmd.log.WriteString(logrus.WarnLevel, "\n")

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
	client, err = kubectl.CheckKubeContext(client, localCache, globalFlags.NoWarn, globalFlags.SwitchContext, cmd.log)
	if err != nil {
		return nil, err
	}

	// load config
	configInterface, err := configLoader.LoadWithCache(ctx, localCache, client, configOptions, cmd.log)
	if err != nil {
		return nil, err
	}

	// adjust config
	err = cmd.adjustConfig(configInterface)
	if err != nil {
		return nil, err
	}

	// create devspace context
	return devspacecontext.NewContext(ctx, cmd.log).
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
	err = pipe.Run(ctx.WithLogger(log.NewStreamLoggerWithFormat(stdoutWriter, stderrWriter, ctx.Log.GetLevel(), log.TimeFormat)))
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

func (cmd *RunPipelineCmd) adjustConfig(conf config.Config) error {
	// check if terminal is enabled
	c := conf.Config()
	if cmd.Terminal {
		if len(c.Dev) == 0 {
			return errors.New("No dev config available in DevSpace config")
		}

		devNames := make([]string, 0, len(c.Dev))
		for k := range c.Dev {
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

		// adjust dev config
		for k := range c.Dev {
			if k == devName {
				if c.Dev[devName].Terminal == nil {
					c.Dev[devName].Terminal = &latest.Terminal{}
				}
				c.Dev[devName].Terminal.Enabled = ptr.Bool(true)
			} else {
				c.Dev[devName].Terminal = nil
			}
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
