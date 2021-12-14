package cmd

import (
	"context"
	"fmt"
	runtimevar "github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	"github.com/loft-sh/devspace/pkg/devspace/imageselector"

	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/services/inject"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/ptr"

	"github.com/loft-sh/devspace/cmd/flags"
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
	configOptions := cmd.ToConfigOptions(cmd.log)
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(cmd.log)
	if err != nil {
		return err
	} else if !configExists || cmd.Pod != "" || cmd.LabelSelector != "" || cmd.Pick {
		client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
		if err != nil {
			return errors.Wrap(err, "create kube client")
		}

		return restartContainer(client, targetselector.NewOptionsFromFlags(cmd.Container, cmd.LabelSelector, cmd.Namespace, cmd.Pod, cmd.Pick), cmd.log)
	}

	log.StartFileLogging()

	// Get config with adjusted cluster config
	generatedConfig, err := configLoader.LoadGenerated(configOptions)
	if err != nil {
		return err
	}
	configOptions.GeneratedConfig = generatedConfig

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, cmd.log)
	if err != nil {
		return err
	}

	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return errors.Wrap(err, "create kube client")
	}

	// If the current kube context or namespace is different than old,
	// show warnings and reset kube client if necessary
	client, err = client.CheckKubeContext(generatedConfig, cmd.NoWarn, cmd.log)
	if err != nil {
		return err
	}

	configOptions.KubeClient = client

	// Get config with adjusted cluster config
	configInterface, err := configLoader.Load(configOptions, cmd.log)
	if err != nil {
		return err
	}
	config := configInterface.Config()

	// Execute plugin hook
	err = hook.ExecuteHooks(client, configInterface, nil, nil, cmd.log, "restart")
	if err != nil {
		return err
	}

	// Resolve dependencies
	dep, err := f.NewDependencyManager(configInterface, client, configOptions, cmd.log).ResolveAll(dependency.ResolveOptions{
		UpdateDependencies: false,
		Verbose:            false,
	})
	if err != nil {
		cmd.log.Warnf("Error resolving dependencies: %v", err)
	}

	// restart containers
	restarts := 0
	for _, syncPath := range config.Dev.Sync {
		if syncPath.OnUpload == nil || !syncPath.OnUpload.RestartContainer {
			continue
		} else if cmd.Name != "" && syncPath.Name != cmd.Name {
			continue
		}

		// create target selector options
		options := targetselector.NewOptionsFromFlags("", "", cmd.Namespace, "", cmd.Pick).ApplyConfigParameter(syncPath.LabelSelector, syncPath.Namespace, syncPath.ContainerName, "")
		options.ImageSelector = []imageselector.ImageSelector{}
		if syncPath.ImageSelector != "" {
			imageSelector, err := runtimevar.NewRuntimeResolver(true).FillRuntimeVariablesAsImageSelector(syncPath.ImageSelector, configInterface, dep)
			if err != nil {
				return err
			}

			options.ImageSelector = append(options.ImageSelector, *imageSelector)
		}

		err = restartContainer(client, options, cmd.log)
		if err != nil {
			return err
		}
		restarts++
	}

	err = configLoader.SaveGenerated(generatedConfig)
	if err != nil {
		cmd.log.Errorf("Error saving generated.yaml: %v", err)
	}

	if restarts == 0 {
		cmd.log.Warn("No containers to restart found, please make sure you have set `dev.sync[*].onUpload.restartContainer` to `true` somewhere in your sync path")
	}
	return nil
}

func restartContainer(client kubectl.Client, options targetselector.Options, log log.Logger) error {
	options.Wait = ptr.Bool(false)
	container, err := targetselector.NewTargetSelector(client).SelectSingleContainer(context.TODO(), options, log)
	if err != nil {
		return errors.Errorf("Error selecting pod: %v", err)
	}

	err = inject.InjectDevSpaceHelper(client, container.Pod, container.Container.Name, "", log)
	if err != nil {
		return errors.Wrap(err, "inject devspace helper")
	}

	stdOut, stdErr, err := client.ExecBuffered(container.Pod, container.Container.Name, []string{inject.DevSpaceHelperContainerPath, "restart"}, nil)
	if err != nil {
		return fmt.Errorf("error restarting container %s in pod %s/%s: %s %s => %v", container.Container.Name, container.Pod.Namespace, container.Pod.Name, string(stdOut), string(stdErr), err)
	}

	log.Donef("Successfully restarted container %s in pod %s/%s", container.Container.Name, container.Pod.Namespace, container.Pod.Name)
	return nil
}
