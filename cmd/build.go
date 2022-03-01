package cmd

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/build/types"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"strings"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/factory"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/mgutz/ansi"

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
			return cmd.Run(f)
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
func (cmd *BuildCmd) Run(f factory.Factory) error {
	// Set config root
	log := f.GetLog()
	configOptions := cmd.ToConfigOptions()
	configLoader, err := f.NewConfigLoader(cmd.ConfigPath)
	if err != nil {
		return err
	}
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Start file logging
	logpkg.StartFileLogging()

	// create kubectl client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace)
	if err != nil {
		log.Warnf("Unable to create new kubectl client: %v", err)
	}

	// Get the config
	configInterface, err := configLoader.Load(context.Background(), client, configOptions, log)
	if err != nil {
		return err
	}

	// create context
	ctx := devspacecontext.NewContext(context.Background(), log).
		WithConfig(configInterface).
		WithKubeClient(client)

	return runWithHooks(ctx, "buildCommand", func() error {
		return cmd.runCommand(ctx, f, configLoader, configOptions)
	})
}

func (cmd *BuildCmd) runCommand(ctx *devspacecontext.Context, f factory.Factory, configLoader loader.ConfigLoader, configOptions *loader.ConfigOptions) error {
	// Force tag
	if len(cmd.Tags) > 0 {
		for _, imageConfig := range ctx.Config.Config().Images {
			imageConfig.Tags = cmd.Tags
		}
	}

	// Dependencies
	dependencies, err := f.NewDependencyManager(ctx, configOptions).BuildAll(ctx, dependency.BuildOptions{
		Dependencies:     cmd.Dependency,
		SkipDependencies: cmd.SkipDependency,
		Verbose:          cmd.VerboseDependencies,

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
	ctx = ctx.WithDependencies(dependencies)

	// Execute plugin hook
	err = hook.ExecuteHooks(ctx, nil, "build")
	if err != nil {
		return err
	}

	// Build images if necessary
	if len(cmd.Dependency) == 0 {
		if len(ctx.Config.Config().Images) > 0 {
			err := f.NewBuildController().Build(ctx, nil, &build.Options{
				SkipPush:                  cmd.SkipPush,
				SkipPushOnLocalKubernetes: cmd.SkipPushLocalKubernetes,
				ForceRebuild:              cmd.ForceBuild,
				Sequential:                cmd.BuildSequential,
				MaxConcurrentBuilds:       cmd.MaxConcurrentBuilds,
			})
			if err != nil {
				if strings.Contains(err.Error(), "no space left on device") {
					return errors.Errorf("Error building image: %v\n\n Try running `%s` to free docker daemon space and retry", err, ansi.Color("devspace cleanup images", "white+b"))
				}

				return errors.Wrap(err, "build images")
			}

			// merge built images
			alreadyBuiltImages, ok := ctx.Config.GetRuntimeVariable(constants.BuiltImagesKey)
			if ok {
				alreadyBuiltImagesMap, ok := alreadyBuiltImages.(map[string]types.ImageNameTag)
				if ok && len(alreadyBuiltImagesMap) > 0 {
					ctx.Log.Donef("Successfully built %d images", len(alreadyBuiltImagesMap))
				} else {
					ctx.Log.Info("No images to rebuild. Run with -b to force rebuilding")
				}
			} else {
				ctx.Log.Info("No images to rebuild. Run with -b to force rebuilding")
			}
		} else {
			ctx.Log.Info("No images defined for this profile")
		}
	} else {
		ctx.Log.Donef("Successfully built images for dependencies: %s", strings.Join(cmd.Dependency, " "))
	}

	return nil
}
