package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	runtimevar "github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/services/inject"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

// RestartCmd holds the required data for the cmd
type RestartCmd struct {
	*flags.GlobalFlags

	Container     string
	Pod           string
	Pick          bool
	LabelSelector string
	Name          string

	log log.Logger
}

// NewRestartCmd creates a new purge command
func NewRestartCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &RestartCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	restartCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restarts containers where the sync restart helper is injected",
		Long: `
#######################################################
################## devspace restart ###################
#######################################################
Restarts containers where the sync restart helper
is injected:

devspace restart
devspace restart -n my-namespace
#######################################################`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f)
		},
	}
	restartCmd.Flags().StringVarP(&cmd.Container, "container", "c", "", "Container name within pod to restart")
	restartCmd.Flags().StringVar(&cmd.Pod, "pod", "", "Pod to restart")
	restartCmd.Flags().StringVarP(&cmd.LabelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")
	restartCmd.Flags().StringVar(&cmd.Name, "name", "", "The sync path name to restart")
	restartCmd.Flags().BoolVar(&cmd.Pick, "pick", true, "Select a pod")

	return restartCmd
}

// Run executes the purge command logic
func (cmd *RestartCmd) Run(f factory.Factory) error {
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
	} else if !configExists || cmd.Pod != "" || cmd.LabelSelector != "" || cmd.Pick {
		client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace)
		if err != nil {
			return errors.Wrap(err, "create kube client")
		}

		// Create context
		ctx := devspacecontext.NewContext(context.Background(), nil, cmd.log).
			WithKubeClient(client)

		return restartContainer(ctx, targetselector.NewOptionsFromFlags(cmd.Container, cmd.LabelSelector, nil, cmd.Namespace, cmd.Pod).WithPick(cmd.Pick))
	}

	log.StartFileLogging()

	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace)
	if err != nil {
		return errors.Wrap(err, "create kube client")
	}

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
	config, err := configLoader.LoadWithCache(context.Background(), localCache, client, configOptions, cmd.log)
	if err != nil {
		return err
	}

	// Create context
	ctx := devspacecontext.NewContext(context.Background(), config.Variables(), cmd.log).
		WithConfig(config).
		WithKubeClient(client)

	// Execute plugin hook
	err = hook.ExecuteHooks(ctx, nil, "restart")
	if err != nil {
		return err
	}

	// Resolve dependencies
	dep, err := f.NewDependencyManager(ctx, configOptions).ResolveAll(ctx, dependency.ResolveOptions{})
	if err != nil {
		cmd.log.Warnf("Error resolving dependencies: %v", err)
	}
	ctx = ctx.WithDependencies(dep)

	// restart containers
	restarts := 0
	for _, devPod := range ctx.Config().Config().Dev {
		if cmd.Name != "" && devPod.Name != cmd.Name {
			continue
		}

		// has sync config
		found := false
		loader.EachDevContainer(devPod, func(devContainer *latest.DevContainer) bool {
			for _, s := range devContainer.Sync {
				if s.OnUpload != nil && s.OnUpload.RestartContainer {
					found = true
					return false
				}
			}
			return true
		})
		if !found {
			continue
		}

		// find containers to restart
		if cmd.Container == "" {
			cmd.Container = devPod.Container
		}

		// create target selector options
		var imageSelector []string
		if devPod.ImageSelector != "" {
			imageSelectorObject, err := runtimevar.NewRuntimeResolver(ctx.WorkingDir(), true).FillRuntimeVariablesAsImageSelector(ctx.Context(), devPod.ImageSelector, ctx.Config(), ctx.Dependencies())
			if err != nil {
				return err
			}

			imageSelector = []string{imageSelectorObject.Image}
		}

		options := targetselector.NewOptionsFromFlags("", "", nil, cmd.Namespace, "").
			WithPick(cmd.Pick).
			ApplyConfigParameter(cmd.Container, devPod.LabelSelector, imageSelector, devPod.Namespace, "")
		err = restartContainer(ctx, options)
		if err != nil {
			return err
		}
		restarts++
	}

	if restarts == 0 {
		cmd.log.Warn("No containers to restart found, please make sure you have set `dev.sync[*].onUpload.restartContainer` to `true` somewhere in your sync path")
	}
	return nil
}

func restartContainer(ctx devspacecontext.Context, options targetselector.Options) error {
	options = options.WithWait(false)
	container, err := targetselector.NewTargetSelector(options).SelectSingleContainer(ctx.Context(), ctx.KubeClient(), ctx.Log())
	if err != nil {
		return errors.Errorf("Error selecting pod: %v", err)
	}

	err = inject.InjectDevSpaceHelper(ctx.Context(), ctx.KubeClient(), container.Pod, container.Container.Name, "", ctx.Log())
	if err != nil {
		return errors.Wrap(err, "inject devspace helper")
	}

	stdOut, stdErr, err := ctx.KubeClient().ExecBuffered(ctx.Context(), container.Pod, container.Container.Name, []string{inject.DevSpaceHelperContainerPath, "restart"}, nil)
	if err != nil {
		return fmt.Errorf("error restarting container %s in pod %s/%s: %s %s => %v", container.Container.Name, container.Pod.Namespace, container.Pod.Name, string(stdOut), string(stdErr), err)
	}

	ctx.Log().Donef("Successfully restarted container %s in pod %s/%s", container.Container.Name, container.Pod.Namespace, container.Pod.Name)
	return nil
}
