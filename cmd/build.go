package cmd

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/factory"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/mgutz/ansi"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// BuildCmd is a struct that defines a command call for "build"
type BuildCmd struct {
	*flags.GlobalFlags

	Tags []string

	SkipPush                bool
	SkipPushLocalKubernetes bool
	VerboseDependencies     bool
	SkipDependency          []string
	Dependency              []string

	ForceBuild          bool
	BuildSequential     bool
	MaxConcurrentBuilds int
	ForceDependencies   bool
}

// NewBuildCmd creates a new devspace build command
func NewBuildCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &BuildCmd{GlobalFlags: globalFlags}

	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Builds all defined images and pushes them",
		Long: `
#######################################################
################## devspace build #####################
#######################################################
Builds all defined images and pushes them
#######################################################`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f, cobraCmd, args)
		},
	}

	buildCmd.Flags().BoolVarP(&cmd.ForceBuild, "force-build", "b", false, "Forces to build every image")
	buildCmd.Flags().BoolVar(&cmd.BuildSequential, "build-sequential", false, "Builds the images one after another instead of in parallel")
	buildCmd.Flags().IntVar(&cmd.MaxConcurrentBuilds, "max-concurrent-builds", 0, "The maximum number of image builds built in parallel (0 for infinite)")

	buildCmd.Flags().BoolVar(&cmd.ForceDependencies, "force-dependencies", true, "Forces to re-evaluate dependencies (use with --force-build --force-deploy to actually force building & deployment of dependencies)")
	buildCmd.Flags().BoolVar(&cmd.VerboseDependencies, "verbose-dependencies", true, "Builds the dependencies verbosely")

	buildCmd.Flags().StringSliceVarP(&cmd.Tags, "tag", "t", []string{}, "Use the given tag for all built images")
	buildCmd.Flags().StringSliceVar(&cmd.SkipDependency, "skip-dependency", []string{}, "Skips building the following dependencies")
	buildCmd.Flags().StringSliceVar(&cmd.Dependency, "dependency", []string{}, "Builds only the specific named dependencies")

	buildCmd.Flags().BoolVar(&cmd.SkipPush, "skip-push", false, "Skips image pushing, useful for minikube deployment")
	buildCmd.Flags().BoolVar(&cmd.SkipPushLocalKubernetes, "skip-push-local-kube", false, "Skips image pushing, if a local kubernetes environment is detected")

	return buildCmd
}

// Run executes the command logic
func (cmd *BuildCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	log := f.GetLog()
	configOptions := cmd.ToConfigOptions(log)
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Start file logging
	logpkg.StartFileLogging()

	// Load config
	generatedConfig, err := configLoader.LoadGenerated(configOptions)
	if err != nil {
		return err
	}
	configOptions.GeneratedConfig = generatedConfig

	// create kubectl client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		log.Warnf("Unable to create new kubectl client: %v", err)
	}
	configOptions.KubeClient = client

	// Get the config
	configInterface, err := configLoader.Load(configOptions, log)
	if err != nil {
		return err
	}
	config := configInterface.Config()

	// Execute plugin hook
	err = plugin.ExecutePluginHook("build")
	if err != nil {
		return err
	}

	// create namespaces if we have a client
	if client != nil {
		err = client.EnsureDeployNamespaces(config, log)
		if err != nil {
			return errors.Errorf("unable to create namespace: %v", err)
		}
	}

	// Force tag
	if len(cmd.Tags) > 0 {
		for _, imageConfig := range config.Images {
			imageConfig.Tags = cmd.Tags
		}
	}

	// Dependencies
	dependencies, err := f.NewDependencyManager(configInterface, client, configOptions, log).BuildAll(dependency.BuildOptions{
		Dependencies:            cmd.Dependency,
		SkipDependencies:        cmd.SkipDependency,
		ForceDeployDependencies: cmd.ForceDependencies,
		Verbose:                 cmd.VerboseDependencies,

		BuildOptions: build.Options{
			SkipPush:                  cmd.SkipPush,
			SkipPushOnLocalKubernetes: cmd.SkipPushLocalKubernetes,
			ForceRebuild:              cmd.ForceBuild,
			Sequential:                cmd.BuildSequential,
			MaxConcurrentBuilds:       cmd.MaxConcurrentBuilds,
		},
	})
	if err != nil {
		return errors.Wrap(err, "build dependencies")
	}

	// Build images if necessary
	if len(cmd.Dependency) == 0 {
		builtImages, err := f.NewBuildController(configInterface, dependencies, client).Build(&build.Options{
			SkipPush:                  cmd.SkipPush,
			SkipPushOnLocalKubernetes: cmd.SkipPushLocalKubernetes,
			ForceRebuild:              cmd.ForceBuild,
			Sequential:                cmd.BuildSequential,
			MaxConcurrentBuilds:       cmd.MaxConcurrentBuilds,
		}, log)
		if err != nil {
			if strings.Index(err.Error(), "no space left on device") != -1 {
				return errors.Errorf("Error building image: %v\n\n Try running `%s` to free docker daemon space and retry", err, ansi.Color("devspace cleanup images", "white+b"))
			}

			return errors.Wrap(err, "build images")
		}

		// Save config if an image was built
		if len(builtImages) > 0 {
			err := configLoader.SaveGenerated(generatedConfig)
			if err != nil {
				return err
			}

			log.Donef("Successfully built %d images", len(builtImages))
		} else {
			log.Info("No images to rebuild. Run with -b to force rebuilding")
		}
	} else {
		log.Donef("Successfully built images for dependencies: %s", strings.Join(cmd.Dependency, " "))
	}

	return nil
}
