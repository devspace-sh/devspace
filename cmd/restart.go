package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/services"
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

	log log.Logger
}

// NewRestartCmd creates a new purge command
func NewRestartCmd(f factory.Factory, globalFlags *flags.GlobalFlags, plugins []plugin.Metadata) *cobra.Command {
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
			return cmd.Run(f, plugins, cobraCmd, args)
		},
	}
	restartCmd.Flags().StringVarP(&cmd.Container, "container", "c", "", "Container name within pod to restart")
	restartCmd.Flags().StringVar(&cmd.Pod, "pod", "", "Pod to restart")
	restartCmd.Flags().StringVarP(&cmd.LabelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")
	restartCmd.Flags().BoolVar(&cmd.Pick, "pick", true, "Select a pod")

	return restartCmd
}

// Run executes the purge command logic
func (cmd *RestartCmd) Run(f factory.Factory, plugins []plugin.Metadata, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	cmd.log = f.GetLog()
	configOptions := cmd.ToConfigOptions()
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
	configOptions.KubeClient = client

	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, cmd.log)
	if err != nil {
		return err
	}

	// Get config with adjusted cluster config
	configInterface, err := configLoader.Load(configOptions, cmd.log)
	if err != nil {
		return err
	}
	config := configInterface.Config()

	// Execute plugin hook
	err = plugin.ExecutePluginHook(plugins, cobraCmd, args, "restart", client.CurrentContext(), client.Namespace(), config)
	if err != nil {
		return err
	}

	// restart containers
	restarts := 0
	if config.Dev != nil {
		for _, syncPath := range config.Dev.Sync {
			if syncPath.OnUpload == nil || !syncPath.OnUpload.RestartContainer {
				continue
			}

			// create target selector options
			options := targetselector.NewOptionsFromFlags("", "", cmd.Namespace, "", cmd.Pick).ApplyConfigParameter(syncPath.LabelSelector, syncPath.Namespace, syncPath.ContainerName, "")
			options.ImageSelector = targetselector.ImageSelectorFromConfig(syncPath.ImageName, config, generatedConfig.GetActive())

			err = restartContainer(client, options, cmd.log)
			if err != nil {
				return err
			}
			restarts++
		}
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

	err = services.InjectDevSpaceHelper(client, container.Pod, container.Container.Name, log)
	if err != nil {
		return errors.Wrap(err, "inject devspace helper")
	}

	stdOut, stdErr, err := client.ExecBuffered(container.Pod, container.Container.Name, []string{services.DevSpaceHelperContainerPath, "restart"}, nil)
	if err != nil {
		return fmt.Errorf("error restarting container %s in pod %s/%s: %s %s => %v", container.Container.Name, container.Pod.Namespace, container.Pod.Name, string(stdOut), string(stdErr), err)
	}

	log.Donef("Successfully restarted container %s in pod %s/%s", container.Container.Name, container.Pod.Namespace, container.Pod.Name)
	return nil
}
