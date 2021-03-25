package cmd

import (
	"fmt"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// LogsCmd holds the logs cmd flags
type LogsCmd struct {
	*flags.GlobalFlags

	LabelSelector string
	Image         string
	Container     string
	Pod           string
	Pick          bool

	Follow            bool
	Wait              bool
	LastAmountOfLines int
}

// NewLogsCmd creates a new login command
func NewLogsCmd(f factory.Factory, globalFlags *flags.GlobalFlags, plugins []plugin.Metadata) *cobra.Command {
	cmd := &LogsCmd{GlobalFlags: globalFlags}

	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "Prints the logs of a pod and attaches to it",
		Long: `
#######################################################
#################### devspace logs ####################
#######################################################
Logs prints the last log of a pod container and attachs 
to it

Example:
devspace logs
devspace logs --namespace=mynamespace
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunLogs(f, plugins, cobraCmd, args)
		},
	}

	logsCmd.Flags().StringVarP(&cmd.Container, "container", "c", "", "Container name within pod where to execute command")
	logsCmd.Flags().StringVar(&cmd.Pod, "pod", "", "Pod to print the logs of")
	logsCmd.Flags().StringVarP(&cmd.LabelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")
	logsCmd.Flags().StringVar(&cmd.Image, "image", "", "Image is the config name of an image to select in the devspace config (e.g. 'default'), it is NOT a docker image like myuser/myimage")
	logsCmd.Flags().BoolVar(&cmd.Pick, "pick", true, "Select a pod")
	logsCmd.Flags().BoolVarP(&cmd.Follow, "follow", "f", false, "Attach to logs afterwards")
	logsCmd.Flags().IntVar(&cmd.LastAmountOfLines, "lines", 200, "Max amount of lines to print from the last log")
	logsCmd.Flags().BoolVar(&cmd.Wait, "wait", false, "Wait for the pod(s) to start if they are not running")

	return logsCmd
}

// RunLogs executes the functionality devspace logs
func (cmd *LogsCmd) RunLogs(f factory.Factory, plugins []plugin.Metadata, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	log := f.GetLog()
	configOptions := cmd.ToConfigOptions()
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return err
	}

	// Load config if possible
	var generatedConfig *generated.Config
	if configExists {
		generatedConfig, err = configLoader.LoadGenerated(configOptions)
		if err != nil {
			return err
		}
		configOptions.GeneratedConfig = generatedConfig
	}

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, log)
	if err != nil {
		return err
	}

	// Get kubectl client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return errors.Wrap(err, "create kube client")
	}

	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, log)
	if err != nil {
		return err
	}

	// Execute plugin hook
	err = plugin.ExecutePluginHook(plugins, cobraCmd, args, "logs", client.CurrentContext(), client.Namespace(), nil)
	if err != nil {
		return err
	}

	// Build options
	options := targetselector.NewOptionsFromFlags(cmd.Container, cmd.LabelSelector, cmd.Namespace, cmd.Pod, cmd.Pick)

	// get image selector if specified
	imageSelector, err := getImageSelector(configLoader, configOptions, cmd.Image, log)
	if err != nil {
		return err
	}

	// set image selector
	options.ImageSelector = imageSelector

	// Start terminal
	err = f.NewServicesClient(nil, generatedConfig, client, log).StartLogs(options, cmd.Follow, int64(cmd.LastAmountOfLines), cmd.Wait)
	if err != nil {
		return err
	}

	return nil
}

func getImageSelector(configLoader loader.ConfigLoader, configOptions *loader.ConfigOptions, image string, log log.Logger) ([]string, error) {
	var imageSelector []string
	if image != "" {
		if configLoader.Exists() == false {
			return nil, errors.New(message.ConfigNotFound)
		}

		config, err := configLoader.Load(configOptions, log)
		if err != nil {
			return nil, err
		}

		imageSelector = targetselector.ImageSelectorFromConfig(image, config.Config(), config.Generated().GetActive())
		if len(imageSelector) == 0 {
			return nil, fmt.Errorf("couldn't find an image with name %s in devspace config", image)
		}
	}

	return imageSelector, nil
}
