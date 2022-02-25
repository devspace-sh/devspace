package cmd

import (
	"context"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"io"
	"os"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	fakekube "github.com/loft-sh/devspace/pkg/devspace/kubectl/testing"
	"github.com/loft-sh/devspace/pkg/util/factory"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// RenderCmd is a struct that defines a command call for "render"
type RenderCmd struct {
	*flags.GlobalFlags

	Tags []string

	SkipPush                bool
	SkipPushLocalKubernetes bool
	VerboseDependencies     bool

	SkipBuild           bool
	ForceBuild          bool
	BuildSequential     bool
	MaxConcurrentBuilds int

	Deployments string

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
	renderCmd.Flags().BoolVar(&cmd.VerboseDependencies, "verbose-dependencies", false, "Builds the dependencies verbosely")
	renderCmd.Flags().StringSliceVarP(&cmd.Tags, "tag", "t", []string{}, "Use the given tag for all built images")
	renderCmd.Flags().BoolVar(&cmd.SkipPush, "skip-push", false, "Skips image pushing, useful for minikube deployment")
	renderCmd.Flags().BoolVar(&cmd.SkipPushLocalKubernetes, "skip-push-local-kube", true, "Skips image pushing, if a local kubernetes environment is detected")
	renderCmd.Flags().BoolVar(&cmd.SkipBuild, "skip-build", false, "Skips image building")
	renderCmd.Flags().StringVar(&cmd.Deployments, "deployments", "", "Only deploy a specific deployment (You can specify multiple deployments comma-separated")

	renderCmd.Flags().BoolVar(&cmd.SkipDependencies, "skip-dependencies", false, "Skips rendering the dependencies")
	renderCmd.Flags().StringSliceVar(&cmd.SkipDependency, "skip-dependency", []string{}, "Skips rendering the following dependencies")
	renderCmd.Flags().StringSliceVar(&cmd.Dependency, "dependency", []string{}, "Renders only the specific named dependencies")

	return renderCmd
}

// Run executes the command logic
func (cmd *RenderCmd) Run(f factory.Factory) error {
	// Set config root
	log := f.GetLog()
	if cmd.Silent {
		log = logpkg.Discard
	}

	configOptions := cmd.ToConfigOptions()
	configLoader, err := loader.NewConfigLoader(cmd.ConfigPath)
	if err != nil {
		return err
	}
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return err
	} else if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Start file logging
	logpkg.StartFileLogging()

	// Create kubectl client and switch context if specified
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace)
	if err != nil {
		log.Warnf("Unable to create new kubectl client: %v", err)
		log.Warn("Using fake client to render resources")
		log.WriteString("\n")
		kube := fake.NewSimpleClientset()
		client = &fakekube.Client{
			Client: kube,
		}
	}

	// Get the config
	config, err := configLoader.Load(client, configOptions, log)
	if err != nil {
		cause := errors.Cause(err)
		if _, ok := cause.(logpkg.SurveyError); ok {
			return errors.New("Cannot load config, because questions for variables are not possible in silent mode. Please set '--show-logs' to true to disable silent mode")
		}

		return err
	}

	// Force tag
	if len(cmd.Tags) > 0 {
		for _, imageConfig := range config.Config().Images {
			imageConfig.Tags = cmd.Tags
		}
	}

	// Create context
	ctx := devspacecontext.NewContext(context.Background(), log).
		WithConfig(config).
		WithKubeClient(client)

	// Render dependencies
	var dependencies []types.Dependency
	if !cmd.SkipDependencies {
		dependencies, err = f.NewDependencyManager(ctx, configOptions).RenderAll(ctx, dependency.RenderOptions{
			Dependencies:     cmd.Dependency,
			SkipDependencies: cmd.SkipDependency,
			SkipBuild:        cmd.SkipBuild,
			Verbose:          cmd.VerboseDependencies,
			Writer:           cmd.Writer,

			BuildOptions: build.Options{
				SkipPush:                  cmd.SkipPush,
				SkipPushOnLocalKubernetes: cmd.SkipPushLocalKubernetes,
				ForceRebuild:              cmd.ForceBuild,
				Sequential:                cmd.BuildSequential,
				MaxConcurrentBuilds:       cmd.MaxConcurrentBuilds,
			},
		})
		if err != nil {
			return errors.Wrap(err, "render dependencies")
		}

	} else {
		dependencies, err = f.NewDependencyManager(ctx, configOptions).ResolveAll(ctx, dependency.ResolveOptions{
			Silent: true,
		})
	}
	if len(cmd.Dependency) > 0 {
		return nil
	}

	// add dependencies to context
	ctx = ctx.WithDependencies(dependencies)

	// Execute plugin hook
	err = hook.ExecuteHooks(ctx, nil, "render")
	if err != nil {
		return err
	}

	// Build images if necessary
	if !cmd.SkipBuild {
		err = f.NewBuildController().Build(ctx, nil, &build.Options{
			SkipPush:                  cmd.SkipPush,
			SkipPushOnLocalKubernetes: cmd.SkipPushLocalKubernetes,
			ForceRebuild:              cmd.ForceBuild,
			MaxConcurrentBuilds:       cmd.MaxConcurrentBuilds,
			Sequential:                cmd.BuildSequential,
		})
		if err != nil {
			if strings.Contains(err.Error(), "no space left on device") {
				return errors.Errorf("Error building image: %v\n\n Try running `%s` to free docker daemon space and retry", err, ansi.Color("devspace cleanup images", "white+b"))
			}

			return errors.Wrap(err, "build images")
		}
	}

	// What deployments should be deployed
	deployments := []string{}
	if cmd.Deployments != "" {
		deployments = strings.Split(cmd.Deployments, ",")
		for index := range deployments {
			deployments[index] = strings.TrimSpace(deployments[index])
		}
	}

	// Deploy all defined deployments
	err = f.NewDeployController().Render(ctx, deployments, &deploy.Options{}, cmd.Writer)
	if err != nil {
		return err
	}

	return nil
}
