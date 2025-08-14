package reset

import (
	"context"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/devspace/services/podreplace"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type podsCmd struct {
	*flags.GlobalFlags

	Force bool

	log log.Logger
}

func newPodsCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &podsCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	podsCmd := &cobra.Command{
		Use:   "pods",
		Short: "Resets the replaced pods",
		Long: `
#######################################################
############### devspace reset pods ###################
#######################################################
Resets the replaced pods to its original state

Examples:
devspace reset pods
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunResetPods(f, cobraCmd, args)
		}}

	podsCmd.Flags().BoolVar(&cmd.Force, "force", false, "If true will force resetting pods even though they might be still used by other DevSpace projects")
	return podsCmd
}

// RunResetPods executes the reset pods command logic
func (cmd *podsCmd) RunResetPods(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
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

	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace)
	if err != nil {
		return errors.Wrap(err, "create kube client")
	}

	// Get config with adjusted cluster config
	localCache, err := configLoader.LoadLocalCache()
	if err != nil {
		return err
	}

	// If the current kube context or namespace is different from old,
	// show warnings and reset kube client if necessary
	client, err = kubectl.CheckKubeContext(client, localCache, cmd.NoWarn, cmd.SwitchContext, false, cmd.log)
	if err != nil {
		return err
	}

	// Get config with adjusted cluster config
	conf, err := configLoader.LoadWithCache(context.Background(), localCache, client, configOptions, cmd.log)
	if err != nil {
		return err
	}

	// create devspace context
	ctx := devspacecontext.NewContext(context.Background(), conf.Variables(), cmd.log).
		WithConfig(conf).
		WithKubeClient(client)

	// Resolve dependencies
	dependencies, err := f.NewDependencyManager(ctx, configOptions).ResolveAll(ctx, dependency.ResolveOptions{})
	if err != nil {
		cmd.log.Warnf("Error resolving dependencies: %v", err)
	}
	ctx = ctx.WithDependencies(dependencies)

	// reset the pods
	ResetPods(ctx, true, cmd.Force)
	return nil
}

// ResetPods deletes the pods created by dev.replacePods
func ResetPods(ctx devspacecontext.Context, dependencies, force bool) {
	resetted := ResetPodsRecursive(ctx, dependencies, force)
	if resetted == 0 {
		ctx.Log().Info("No dev pods to reset found")
	} else {
		ctx.Log().Donef("Successfully reset %d pods", resetted)
	}
}

func ResetPodsRecursive(ctx devspacecontext.Context, dependencies, force bool) int {
	resetted := 0
	if dependencies {
		for _, d := range ctx.Dependencies() {
			resetted += ResetPodsRecursive(ctx.AsDependency(d), dependencies, force)
		}
	}

	// create pod replacer
	podReplacer := podreplace.NewPodReplacer()
	for _, replacePodCache := range ctx.Config().RemoteCache().ListDevPods() {
		deleted, err := podReplacer.RevertReplacePod(ctx, &replacePodCache, &deploy.PurgeOptions{ForcePurge: force})
		if err != nil {
			ctx.Log().Warnf("Error resetting replaced pod: %v", err)
		} else if deleted {
			resetted++
		}
	}

	return resetted
}
