package cmd

import (
	"context"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/build"
	"github.com/devspace-cloud/devspace/pkg/devspace/dependency"
	"github.com/mgutz/ansi"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// BuildCmd is a struct that defines a command call for "up"
type BuildCmd struct {
	SkipPush                bool
	AllowCyclicDependencies bool

	ForceBuild        bool
	BuildSequential   bool
	ForceDependencies bool
}

// NewBuildCmd creates a new devspace build command
func NewBuildCmd() *cobra.Command {
	cmd := &BuildCmd{}

	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Builds all defined images and pushes them",
		Long: `
#######################################################
################## devspace build #####################
#######################################################
Builds all defined images and pushes them
#######################################################`,
		Run: cmd.Run,
	}

	buildCmd.Flags().BoolVar(&cmd.AllowCyclicDependencies, "allow-cyclic", false, "When enabled allows cyclic dependencies")

	buildCmd.Flags().BoolVarP(&cmd.ForceBuild, "force-build", "b", false, "Forces to build every image")
	buildCmd.Flags().BoolVar(&cmd.BuildSequential, "build-sequential", false, "Builds the images one after another instead of in parallel")
	buildCmd.Flags().BoolVar(&cmd.ForceDependencies, "force-dependencies", false, "Forces to re-evaluate dependencies (use with --force-build --force-deploy to actually force building & deployment of dependencies)")

	buildCmd.Flags().BoolVar(&cmd.SkipPush, "skip-push", false, "Skips image pushing, useful for minikube deployment")

	return buildCmd
}

// Run executes the command logic
func (cmd *BuildCmd) Run(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	// Start file logging
	log.StartFileLogging()

	// Load config
	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading generated.yaml: %v", err)
	}

	// Get the config
	config := configutil.GetConfig(context.Background())

	// Dependencies
	err = dependency.BuildAll(config, generatedConfig, cmd.AllowCyclicDependencies, false, cmd.SkipPush, cmd.ForceDependencies, cmd.ForceBuild, log.GetInstance())
	if err != nil {
		log.Fatalf("Error deploying dependencies: %v", err)
	}

	// Build images if necessary
	builtImages, err := build.All(config, generatedConfig.GetActive(), nil, cmd.SkipPush, true, cmd.ForceBuild, cmd.BuildSequential, log.GetInstance())
	if err != nil {
		if strings.Index(err.Error(), "no space left on device") != -1 {
			log.Fatalf("Error building image: %v\n\n Try running `%s` to free docker daemon space and retry", err, ansi.Color("devspace cleanup images", "white+b"))
		}

		log.Fatalf("Error building image: %v", err)
	}

	// Save config if an image was built
	if len(builtImages) > 0 {
		err := generated.SaveConfig(generatedConfig)
		if err != nil {
			log.Fatalf("Error saving generated config: %v", err)
		}

		log.Donef("Successfully built %d images", len(builtImages))
	} else {
		log.Info("No images to rebuild. Run with -b to force rebuilding")
	}
}
