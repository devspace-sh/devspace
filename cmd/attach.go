package cmd

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

// AttachCmd is a struct that defines a command call for "enter"
type AttachCmd struct {
	*flags.GlobalFlags

	LabelSelector string
	Image         string
	Container     string
	Pod           string
	Pick          bool
}

// NewAttachCmd creates a new attach command
func NewAttachCmd(f factory.Factory, globalFlags *flags.GlobalFlags, plugins []plugin.Metadata) *cobra.Command {
	cmd := &AttachCmd{GlobalFlags: globalFlags}

	attachCmd := &cobra.Command{
		Use:   "attach",
		Short: "Attaches to a container",
		Long: `
#######################################################
################# devspace attach #####################
#######################################################
Attaches to a running container

devspace attach
devspace attach --pick # Select pod to enter
devspace attach -c my-container
devspace attach -n my-namespace
#######################################################`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, plugins, cobraCmd, args)
		},
	}

	attachCmd.Flags().StringVarP(&cmd.Container, "container", "c", "", "Container name within pod where to execute command")
	attachCmd.Flags().StringVar(&cmd.Pod, "pod", "", "Pod to open a shell to")
	attachCmd.Flags().StringVar(&cmd.Image, "image", "", "Image is the config name of an image to select in the devspace config (e.g. 'default'), it is NOT a docker image like myuser/myimage")
	attachCmd.Flags().StringVarP(&cmd.LabelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")

	attachCmd.Flags().BoolVar(&cmd.Pick, "pick", true, "Select a pod")

	return attachCmd
}

// Run executes the command logic
func (cmd *AttachCmd) Run(f factory.Factory, plugins []plugin.Metadata, cobraCmd *cobra.Command, args []string) error {
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
		return errors.Wrap(err, "new kube client")
	}

	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, log)
	if err != nil {
		return err
	}

	// Execute plugin hook
	err = plugin.ExecutePluginHook(plugins, cobraCmd, args, "attach", client.CurrentContext(), client.Namespace(), nil)
	if err != nil {
		return err
	}

	// Build params
	options := targetselector.NewOptionsFromFlags(cmd.Container, cmd.LabelSelector, cmd.Namespace, cmd.Pod, cmd.Pick)

	// get image selector if specified
	imageSelector, err := getImageSelector(configLoader, configOptions, cmd.Image, log)
	if err != nil {
		return err
	}

	// set image selector
	options.ImageSelector = imageSelector
	options.Wait = ptr.Bool(false)

	// Start attach
	return f.NewServicesClient(nil, nil, client, log).StartAttach(options, make(chan error))
}
