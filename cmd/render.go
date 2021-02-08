package cmd

import (
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"os"
	"strings"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
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
	AllowCyclicDependencies bool
	VerboseDependencies     bool

	SkipBuild       bool
	ForceBuild      bool
	BuildSequential bool

	ShowLogs    bool
	Deployments string

	SkipDependencies bool
	Dependency       []string
}

// NewRenderCmd creates a new devspace render command
func NewRenderCmd(f factory.Factory, globalFlags *flags.GlobalFlags, plugins []plugin.Metadata) *cobra.Command {
	cmd := &RenderCmd{GlobalFlags: globalFlags}

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
			return cmd.Run(f, plugins, cobraCmd, args)
		},
	}

	renderCmd.Flags().BoolVar(&cmd.AllowCyclicDependencies, "allow-cyclic", false, "When enabled allows cyclic dependencies")

	renderCmd.Flags().BoolVarP(&cmd.ForceBuild, "force-build", "b", false, "Forces to build every image")
	renderCmd.Flags().BoolVar(&cmd.BuildSequential, "build-sequential", false, "Builds the images one after another instead of in parallel")
	renderCmd.Flags().BoolVar(&cmd.VerboseDependencies, "verbose-dependencies", false, "Builds the dependencies verbosely")
	renderCmd.Flags().StringSliceVarP(&cmd.Tags, "tag", "t", []string{}, "Use the given tag for all built images")
	renderCmd.Flags().BoolVar(&cmd.ShowLogs, "show-logs", false, "Shows the build logs")
	renderCmd.Flags().BoolVar(&cmd.SkipPush, "skip-push", false, "Skips image pushing, useful for minikube deployment")
	renderCmd.Flags().BoolVar(&cmd.SkipPushLocalKubernetes, "skip-push-local-kube", true, "Skips image pushing, if a local kubernetes environment is detected")
	renderCmd.Flags().BoolVar(&cmd.SkipBuild, "skip-build", false, "Skips image building")
	renderCmd.Flags().StringVar(&cmd.Deployments, "deployments", "", "Only deploy a specifc deployment (You can specify multiple deployments comma-separated")

	renderCmd.Flags().BoolVar(&cmd.SkipDependencies, "skip-dependencies", false, "Skips rendering the dependencies")
	renderCmd.Flags().StringSliceVar(&cmd.Dependency, "dependency", []string{}, "Renders only the specific named dependencies")

	return renderCmd
}

// Run executes the command logic
func (cmd *RenderCmd) Run(f factory.Factory, plugins []plugin.Metadata, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	log := f.GetLog()
	if cmd.ShowLogs == false {
		log = logpkg.Discard
	}

	configOptions := cmd.ToConfigOptions()
	configLoader := loader.NewConfigLoader(configOptions, log)
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	} else if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Start file logging
	logpkg.StartFileLogging()

	// Load config
	generatedConfig, err := configLoader.Generated()
	if err != nil {
		return err
	}

	// Create kubectl client and switch context if specified
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	// Get the config
	config, err := configLoader.RestoreLoadSave(client)
	if err != nil {
		cause := errors.Cause(err)
		if _, ok := cause.(logpkg.SurveyError); ok {
			return errors.New("Cannot load config, because questions for variables are not possible in silent mode. Please set '--show-logs' to true to disable silent mode")
		}

		return err
	}

	// Force tag
	if len(cmd.Tags) > 0 {
		for _, imageConfig := range config.Images {
			imageConfig.Tags = cmd.Tags
		}
	}

	// Execute plugin hook
	err = plugin.ExecutePluginHook(plugins, cobraCmd, args, "render", client.CurrentContext(), client.Namespace(), config)
	if err != nil {
		return err
	}

	// Create Dependencymanager
	if cmd.SkipDependencies == false {
		manager, err := f.NewDependencyManager(config, generatedConfig, client, cmd.AllowCyclicDependencies, configOptions, log)
		if err != nil {
			return errors.Wrap(err, "new manager")
		}

		// Dependencies
		err = manager.RenderAll(dependency.RenderOptions{
			Dependencies: cmd.Dependency,
			SkipPush:     cmd.SkipPush,
			SkipBuild:    cmd.SkipBuild,
			ForceBuild:   cmd.ForceBuild,
			Verbose:      cmd.VerboseDependencies,
		})
		if err != nil {
			return errors.Wrap(err, "render dependencies")
		}
	}

	if len(cmd.Dependency) > 0 {
		return nil
	}

	// Build images if necessary
	builtImages := map[string]string{}
	if cmd.SkipBuild == false {
		builtImages, err = f.NewBuildController(config, generatedConfig.GetActive(), client).Build(&build.Options{
			SkipPush:                  cmd.SkipPush,
			SkipPushOnLocalKubernetes: cmd.SkipPushLocalKubernetes,
			ForceRebuild:              cmd.ForceBuild,
			Sequential:                cmd.BuildSequential,
		}, log)
		if err != nil {
			if strings.Index(err.Error(), "no space left on device") != -1 {
				return errors.Errorf("Error building image: %v\n\n Try running `%s` to free docker daemon space and retry", err, ansi.Color("devspace cleanup images", "white+b"))
			}

			return errors.Wrap(err, "build images")
		}
	}

	// Save config if an image was built
	if len(builtImages) > 0 {
		err := configLoader.SaveGenerated()
		if err != nil {
			return err
		}

		log.Donef("Successfully built %d images", len(builtImages))
	} else {
		log.Info("No images to rebuild. Run with -b to force rebuilding")
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
	err = f.NewDeployController(config, generatedConfig.GetActive(), client).Render(&deploy.Options{
		BuiltImages: builtImages,
		Deployments: deployments,
	}, os.Stdout, log)
	if err != nil {
		return err
	}

	return nil
}
