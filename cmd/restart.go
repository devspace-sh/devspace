package cmd

import (
	"fmt"
	"github.com/devspace-cloud/devspace/pkg/devspace/plugin"
	"github.com/devspace-cloud/devspace/pkg/devspace/services"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/factory"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

// RestartCmd holds the required data for the cmd
type RestartCmd struct {
	*flags.GlobalFlags

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

	return restartCmd
}

// Run executes the purge command logic
func (cmd *RestartCmd) Run(f factory.Factory, plugins []plugin.Metadata, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	cmd.log = f.GetLog()
	configOptions := cmd.ToConfigOptions()
	configLoader := f.NewConfigLoader(configOptions, cmd.log)
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	} else if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	log.StartFileLogging()

	// Get config with adjusted cluster config
	generatedConfig, err := configLoader.Generated()
	if err != nil {
		return err
	}

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, cmd.log)
	if err != nil {
		return err
	}

	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return errors.Wrap(err, "create kube client")
	}

	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, cmd.log)
	if err != nil {
		return err
	}

	// Get config with adjusted cluster config
	config, err := configLoader.Load()
	if err != nil {
		return err
	}

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

			selector, err := targetselector.NewTargetSelector(config, client, &targetselector.SelectorParameter{
				CmdParameter: targetselector.CmdParameter{
					Namespace: cmd.Namespace,
				},
				ConfigParameter: targetselector.ConfigParameter{
					LabelSelector: syncPath.LabelSelector,
					Namespace:     syncPath.Namespace,
					ContainerName: syncPath.ContainerName,
				},
			}, true, targetselector.ImageSelectorFromConfig(syncPath.ImageName, config, generatedConfig))
			if err != nil {
				return errors.Errorf("Error creating target selector: %v", err)
			}

			cmd.log.StartWait("Restart: Waiting for pods...")
			pod, container, err := selector.GetContainer(false, cmd.log)
			cmd.log.StopWait()
			if err != nil {
				return errors.Errorf("Error selecting pod: %v", err)
			}

			err = services.InjectDevSpaceHelper(client, pod, container.Name, cmd.log)
			if err != nil {
				return errors.Wrap(err, "inject devspace helper")
			}

			stdOut, stdErr, err := client.ExecBuffered(pod, container.Name, []string{services.DevSpaceHelperContainerPath, "restart"}, nil)
			if err != nil {
				return fmt.Errorf("error restarting container %s in pod %s/%s: %s %s => %v", container.Name, pod.Namespace, pod.Name, string(stdOut), string(stdErr), err)
			}

			cmd.log.Donef("Successfully restarted container %s in pod %s/%s", container.Name, pod.Namespace, pod.Name)
			restarts++
		}
	}

	err = configLoader.SaveGenerated()
	if err != nil {
		cmd.log.Errorf("Error saving generated.yaml: %v", err)
	}

	if restarts == 0 {
		cmd.log.Warn("No containers to restart found, please make sure you have set `dev.sync[*].onUpload.restartContainer` to `true` somewhere in your sync path")
	}
	return nil
}
