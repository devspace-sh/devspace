package cmd

import (
	"strings"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/build"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/dependency"
	logpkg "github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// BuildCmd is a struct that defines a command call for "up"
type BuildCmd struct {
	*flags.GlobalFlags

	Tag string

	SkipPush                bool
	AllowCyclicDependencies bool
	VerboseDependencies     bool

	ForceBuild        bool
	BuildSequential   bool
	ForceDependencies bool
}

// NewBuildCmd creates a new devspace build command
func NewBuildCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
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
		RunE: cmd.Run,
	}

	buildCmd.Flags().BoolVar(&cmd.AllowCyclicDependencies, "allow-cyclic", false, "When enabled allows cyclic dependencies")

	buildCmd.Flags().BoolVarP(&cmd.ForceBuild, "force-build", "b", false, "Forces to build every image")
	buildCmd.Flags().BoolVar(&cmd.BuildSequential, "build-sequential", false, "Builds the images one after another instead of in parallel")
	buildCmd.Flags().BoolVar(&cmd.ForceDependencies, "force-dependencies", false, "Forces to re-evaluate dependencies (use with --force-build --force-deploy to actually force building & deployment of dependencies)")
	buildCmd.Flags().BoolVar(&cmd.VerboseDependencies, "verbose-dependencies", false, "Builds the dependencies verbosely")
	buildCmd.Flags().StringVarP(&cmd.Tag, "tag", "t", "", "Use the given tag for all built images")

	buildCmd.Flags().BoolVar(&cmd.SkipPush, "skip-push", false, "Skips image pushing, useful for minikube deployment")

	return buildCmd
}

// Run executes the command logic
func (cmd *BuildCmd) Run(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	log := logpkg.GetInstance()
	configOptions := cmd.ToConfigOptions()
	configLoader := loader.NewConfigLoader(configOptions, log)
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Start file logging
	logpkg.StartFileLogging()

	// Load config
	generatedConfig, err := configLoader.Generated()
	if err != nil {
		return err
	}

	// Get the config
	config, err := configLoader.Load()
	if err != nil {
		return err
	}

	// Force tag
	if cmd.Tag != "" {
		for _, imageConfig := range config.Images {
			imageConfig.Tag = cmd.Tag
		}
	}

	// Create Dependencymanager
	manager, err := dependency.NewManager(config, generatedConfig, nil, cmd.AllowCyclicDependencies, configOptions, log)
	if err != nil {
		return errors.Wrap(err, "new manager")
	}

	// Dependencies
	err = manager.BuildAll(dependency.BuildOptions{
		SkipPush:                cmd.SkipPush,
		ForceDeployDependencies: cmd.ForceDependencies,
		ForceBuild:              cmd.ForceBuild,
		Verbose:                 cmd.VerboseDependencies,
	})
	if err != nil {
		return errors.Wrap(err, "build dependencies")
	}

	// Build images if necessary
	builtImages, err := build.NewController(config, generatedConfig.GetActive(), nil).Build(&build.Options{
		SkipPush:     cmd.SkipPush,
		IsDev:        true,
		ForceRebuild: cmd.ForceBuild,
		Sequential:   cmd.BuildSequential,
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

	return nil
}
