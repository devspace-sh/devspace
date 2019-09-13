package cmd

import (
	"strings"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/build"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/dependency"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// BuildCmd is a struct that defines a command call for "up"
type BuildCmd struct {
	*flags.GlobalFlags

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

	buildCmd.Flags().BoolVar(&cmd.SkipPush, "skip-push", false, "Skips image pushing, useful for minikube deployment")

	return buildCmd
}

// Run executes the command logic
func (cmd *BuildCmd) Run(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	// Start file logging
	log.StartFileLogging()

	// Load config
	generatedConfig, err := generated.LoadConfig(cmd.Profile)
	if err != nil {
		return err
	}

	// Get the config
	configOptions := configutil.FromFlags(cmd.GlobalFlags)
	config, err := configutil.GetConfig(configOptions)
	if err != nil {
		return err
	}

	// Dependencies
	err = dependency.BuildAll(config, generatedConfig, cmd.AllowCyclicDependencies, false, cmd.SkipPush, cmd.ForceDependencies, cmd.ForceBuild, cmd.VerboseDependencies, configOptions, log.GetInstance())
	if err != nil {
		return errors.Wrap(err, "build dependencies")
	}

	// Build images if necessary
	builtImages, err := build.All(config, generatedConfig.GetActive(), nil, cmd.SkipPush, true, cmd.ForceBuild, cmd.BuildSequential, false, log.GetInstance())
	if err != nil {
		if strings.Index(err.Error(), "no space left on device") != -1 {
			return errors.Errorf("Error building image: %v\n\n Try running `%s` to free docker daemon space and retry", err, ansi.Color("devspace cleanup images", "white+b"))
		}

		return errors.Wrap(err, "build images")
	}

	// Save config if an image was built
	if len(builtImages) > 0 {
		err := generated.SaveConfig(generatedConfig)
		if err != nil {
			return err
		}

		log.Donef("Successfully built %d images", len(builtImages))
	} else {
		log.Info("No images to rebuild. Run with -b to force rebuilding")
	}

	return nil
}
