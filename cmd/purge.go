package cmd

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"strings"

	"github.com/loft-sh/devspace/cmd/reset"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"

	"github.com/loft-sh/devspace/pkg/util/factory"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

// PurgeCmd holds the required data for the purge cmd
type PurgeCmd struct {
	*flags.GlobalFlags

	Deployments         string
	VerboseDependencies bool
	All                 bool

	SkipDependency []string
	Dependency     []string

	log log.Logger
}

// NewPurgeCmd creates a new purge command
func NewPurgeCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &PurgeCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	purgeCmd := &cobra.Command{
		Use:   "purge",
		Short: "Delete deployed resources",
		Long: `
#######################################################
################### devspace purge ####################
#######################################################
Deletes the deployed kuberenetes resources:

devspace purge
devspace purge --dependencies
devspace purge -d my-deployment
#######################################################`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f)
		},
	}

	purgeCmd.Flags().StringVarP(&cmd.Deployments, "deployments", "d", "", "The deployment to delete (You can specify multiple deployments comma-separated, e.g. devspace-default,devspace-database etc.)")
	purgeCmd.Flags().BoolVarP(&cmd.All, "all", "a", true, "When enabled purges the dependencies as well")
	purgeCmd.Flags().BoolVar(&cmd.VerboseDependencies, "verbose-dependencies", true, "Builds the dependencies verbosely")

	purgeCmd.Flags().StringSliceVar(&cmd.SkipDependency, "skip-dependency", []string{}, "Skips the following dependencies from purging")
	purgeCmd.Flags().StringSliceVar(&cmd.Dependency, "dependency", []string{}, "Purges only the specific named dependencies")
	return purgeCmd
}

// Run executes the purge command logic
func (cmd *PurgeCmd) Run(f factory.Factory) error {
	// Set config root
	cmd.log = f.GetLog()
	configOptions := cmd.ToConfigOptions()
	configLoader, err := f.NewConfigLoader(cmd.ConfigPath)
	if err != nil {
		return err
	}
	configExists, err := configLoader.SetDevSpaceRoot(cmd.log)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	log.StartFileLogging()
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace)
	if err != nil {
		return errors.Wrap(err, "create kube client")
	}

	// Get config with adjusted cluster config
	localCache, err := localcache.NewCacheLoaderFromDevSpacePath(cmd.ConfigPath).Load()
	if err != nil {
		return err
	}

	// If the current kube context or namespace is different than old,
	// show warnings and reset kube client if necessary
	client, err = client.CheckKubeContext(localCache, cmd.NoWarn, cmd.log)
	if err != nil {
		return err
	}

	// Get config with adjusted cluster config
	configInterface, err := configLoader.LoadWithCache(localCache, client, configOptions, cmd.log)
	if err != nil {
		return err
	}

	// create devspace context
	ctx := devspacecontext.NewContext(context.Background(), cmd.log).
		WithConfig(configInterface).
		WithKubeClient(client)

	return runWithHooks(ctx, "purgeCommand", func() error {
		return cmd.runCommand(ctx, f, configOptions)
	})
}

func (cmd *PurgeCmd) runCommand(ctx *devspacecontext.Context, f factory.Factory, configOptions *loader.ConfigOptions) error {
	// Only purge if we don't specify dependency
	if len(cmd.Dependency) == 0 {
		// Resolve dependencies
		dependencies, err := f.NewDependencyManager(ctx, configOptions).ResolveAll(ctx, dependency.ResolveOptions{
			SkipDependencies: cmd.SkipDependency,
			Silent:           true,
			Verbose:          false,
		})
		if err != nil {
			cmd.log.Warnf("Error resolving dependencies: %v", err)
		}
		ctx = ctx.WithDependencies(dependencies)

		// Reset replaced pods
		if len(ctx.Config.RemoteCache().ListDevPods()) > 0 {
			reset.ResetPods(ctx, false)
		}

		deployments := []string{}
		if cmd.Deployments != "" {
			deployments = strings.Split(cmd.Deployments, ",")
			for index := range deployments {
				deployments[index] = strings.TrimSpace(deployments[index])
			}
		}

		// Purge deployments
		err = f.NewDeployController().Purge(ctx, deployments)
		if err != nil {
			cmd.log.Errorf("Error purging deployments: %v", err)
		}
	}

	// Purge dependencies
	if cmd.All || len(cmd.Dependency) > 0 {
		// Resolve dependencies
		dependencies, err := f.NewDependencyManager(ctx, configOptions).ResolveAll(ctx, dependency.ResolveOptions{
			SkipDependencies: cmd.SkipDependency,
			Dependencies:     cmd.Dependency,
			Silent:           true,
			Verbose:          false,
		})
		if err != nil {
			cmd.log.Warnf("Error resolving dependencies: %v", err)
		}
		ctx = ctx.WithDependencies(dependencies)

		// Reset all dev pods
		for _, d := range dependencies {
			reset.ResetPodsRecursive(ctx.AsDependency(d), true)
		}

		// Test
		_, err = f.NewDependencyManager(ctx, configOptions).PurgeAll(ctx, dependency.PurgeOptions{
			SkipDependencies: cmd.SkipDependency,
			Dependencies:     cmd.Dependency,
			Verbose:          cmd.VerboseDependencies,
		})
		if err != nil {
			cmd.log.Errorf("Error purging dependencies: %v", err)
		}
	}

	err := ctx.Config.RemoteCache().Save(ctx.Context, ctx.KubeClient)
	if err != nil {
		cmd.log.Errorf("Error saving generated.yaml: %v", err)
	}

	return nil
}
