package cmd

import (
	"fmt"

	"github.com/loft-sh/devspace/cmd/flags"
	config2 "github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/util"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/imageselector"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// LogsCmd holds the logs cmd flags
type LogsCmd struct {
	*flags.GlobalFlags

	LabelSelector string
	ImageSelector string
	Image         string
	Container     string
	Pod           string
	Pick          bool

	Follow            bool
	Wait              bool
	LastAmountOfLines int
}

// NewLogsCmd creates a new login command
func NewLogsCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
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
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.RunLogs(f)
		},
	}

	logsCmd.Flags().StringVarP(&cmd.Container, "container", "c", "", "Container name within pod where to execute command")
	logsCmd.Flags().StringVar(&cmd.Pod, "pod", "", "Pod to print the logs of")
	logsCmd.Flags().StringVarP(&cmd.LabelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")
	logsCmd.Flags().StringVar(&cmd.ImageSelector, "image-selector", "", "The image to search a pod for (e.g. nginx, nginx:latest, image(app), nginx:tag(app))")
	logsCmd.Flags().StringVar(&cmd.Image, "image", "", "Image is the config name of an image to select in the devspace config (e.g. 'default'), it is NOT a docker image like myuser/myimage")
	logsCmd.Flags().BoolVar(&cmd.Pick, "pick", true, "Select a pod")
	logsCmd.Flags().BoolVarP(&cmd.Follow, "follow", "f", false, "Attach to logs afterwards")
	logsCmd.Flags().IntVar(&cmd.LastAmountOfLines, "lines", 200, "Max amount of lines to print from the last log")
	logsCmd.Flags().BoolVar(&cmd.Wait, "wait", false, "Wait for the pod(s) to start if they are not running")

	return logsCmd
}

// RunLogs executes the functionality devspace logs
func (cmd *LogsCmd) RunLogs(f factory.Factory) error {
	// Set config root
	log := f.GetLog()
	configOptions := cmd.ToConfigOptions(log)
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

	// Execute plugin hook
	err = plugin.ExecutePluginHook("logs")
	if err != nil {
		return err
	}

	// Build options
	options := targetselector.NewOptionsFromFlags(cmd.Container, cmd.LabelSelector, cmd.Namespace, cmd.Pod, cmd.Pick)

	// get image selector if specified
	imageSelector, err := getImageSelector(client, configLoader, configOptions, cmd.Image, cmd.ImageSelector, log)
	if err != nil {
		return err
	}

	// set image selector
	options.ImageSelector = imageSelector

	// Start terminal
	err = f.NewServicesClient(nil, nil, client, log).StartLogs(options, cmd.Follow, int64(cmd.LastAmountOfLines), cmd.Wait)
	if err != nil {
		return err
	}

	return nil
}

func getImageSelector(client kubectl.Client, configLoader loader.ConfigLoader, configOptions *loader.ConfigOptions, image, imageSelector string, log log.Logger) ([]imageselector.ImageSelector, error) {
	var imageSelectors []imageselector.ImageSelector
	if imageSelector != "" {
		var (
			err          error
			config       config2.Config
			dependencies []types.Dependency
		)
		if !configLoader.Exists() {
			config = config2.Ensure(nil)
		} else {
			config, err = configLoader.Load(configOptions, log)
			if err != nil {
				return nil, err
			}

			dependencies, err = dependency.NewManager(config, client, configOptions, log).ResolveAll(dependency.ResolveOptions{
				Silent: true,
			})
			if err != nil {
				log.Warnf("Error resolving dependencies: %v", err)
			}
		}

		resolved, err := util.ResolveImageAsImageSelector(imageSelector, config, dependencies)
		if err != nil {
			return nil, err
		}

		imageSelectors = append(imageSelectors, *resolved)
	} else if image != "" {
		log.Warnf("Flag --image is deprecated, please use --image-selector instead")

		if !configLoader.Exists() {
			return nil, errors.New(message.ConfigNotFound)
		}

		config, err := configLoader.Load(configOptions, log)
		if err != nil {
			return nil, err
		}

		resolved, err := dependency.NewManager(config, client, configOptions, log).ResolveAll(dependency.ResolveOptions{
			Silent: true,
		})
		if err != nil {
			log.Warnf("Error resolving dependencies: %v", err)
		}

		imageSelector, err := imageselector.Resolve(image, config, resolved)
		if err != nil {
			return nil, err
		} else if imageSelector == nil {
			return nil, fmt.Errorf("couldn't find an image with name %s in devspace config", image)
		}

		imageSelectors = append(imageSelectors, *imageSelector)
	}

	return imageSelectors, nil
}
