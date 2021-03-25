package cmd

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/exit"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// EnterCmd is a struct that defines a command call for "enter"
type EnterCmd struct {
	*flags.GlobalFlags

	LabelSelector string
	Image         string
	Container     string
	Pod           string
	Pick          bool
	Wait          bool

	WorkingDirectory string
}

// NewEnterCmd creates a new enter command
func NewEnterCmd(f factory.Factory, globalFlags *flags.GlobalFlags, plugins []plugin.Metadata) *cobra.Command {
	cmd := &EnterCmd{GlobalFlags: globalFlags}

	enterCmd := &cobra.Command{
		Use:   "enter",
		Short: "Open a shell to a container",
		Long: `
#######################################################
################## devspace enter #####################
#######################################################
Execute a command or start a new terminal in your 
devspace:

devspace enter
devspace enter --pick # Select pod to enter
devspace enter bash
devspace enter -c my-container
devspace enter bash -n my-namespace
devspace enter bash -l release=test
#######################################################`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, plugins, cobraCmd, args)
		},
	}

	enterCmd.Flags().StringVarP(&cmd.Container, "container", "c", "", "Container name within pod where to execute command")
	enterCmd.Flags().StringVar(&cmd.Pod, "pod", "", "Pod to open a shell to")
	enterCmd.Flags().StringVar(&cmd.Image, "image", "", "Image is the config name of an image to select in the devspace config (e.g. 'default'), it is NOT a docker image like myuser/myimage")
	enterCmd.Flags().StringVarP(&cmd.LabelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")
	enterCmd.Flags().StringVar(&cmd.WorkingDirectory, "workdir", "", "The working directory where to open the terminal or execute the command")

	enterCmd.Flags().BoolVar(&cmd.Pick, "pick", true, "Select a pod / container if multiple are found")
	enterCmd.Flags().BoolVar(&cmd.Wait, "wait", false, "Wait for the pod(s) to start if they are not running")

	return enterCmd
}

// Run executes the command logic
func (cmd *EnterCmd) Run(f factory.Factory, plugins []plugin.Metadata, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	logger := f.GetLog()
	configOptions := cmd.ToConfigOptions()
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(logger)
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
	err = cmd.UseLastContext(generatedConfig, logger)
	if err != nil {
		return err
	}

	// Get kubectl client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return errors.Wrap(err, "new kube client")
	}

	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, logger)
	if err != nil {
		return err
	}

	// Execute plugin hook
	err = plugin.ExecutePluginHook(plugins, cobraCmd, args, "enter", client.CurrentContext(), client.Namespace(), nil)
	if err != nil {
		return err
	}

	// Build params
	selectorOptions := targetselector.NewOptionsFromFlags(cmd.Container, cmd.LabelSelector, cmd.Namespace, cmd.Pod, cmd.Pick)

	// get image selector if specified
	imageSelector, err := getImageSelector(configLoader, configOptions, cmd.Image, logger)
	if err != nil {
		return err
	}

	// set image selector
	selectorOptions.ImageSelector = imageSelector

	// Start terminal
	exitCode, err := f.NewServicesClient(nil, generatedConfig, client, logger).StartTerminal(selectorOptions, args, cmd.WorkingDirectory, make(chan error), cmd.Wait)
	if err != nil {
		return err
	} else if exitCode != 0 {
		return &exit.ReturnCodeError{
			ExitCode: exitCode,
		}
	}

	return nil
}
