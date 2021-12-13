package cmd

import (
	"io"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
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
	ImageSelector string
	Image         string
	Container     string
	Pod           string
	Pick          bool
	Wait          bool
	Reconnect     bool

	WorkingDirectory string

	// used for testing
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
}

// NewEnterCmd creates a new enter command
func NewEnterCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
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
devspace enter bash --image-selector nginx:latest
devspace enter bash --image-selector "${runtime.images.app.image}:${runtime.images.app.tag}"
#######################################################`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f, cobraCmd, args)
		},
	}

	enterCmd.Flags().StringVarP(&cmd.Container, "container", "c", "", "Container name within pod where to execute command")
	enterCmd.Flags().StringVar(&cmd.Pod, "pod", "", "Pod to open a shell to")
	enterCmd.Flags().StringVar(&cmd.Image, "image", "", "Image is the config name of an image to select in the devspace config (e.g. 'default'), it is NOT a docker image like myuser/myimage")
	enterCmd.Flags().StringVarP(&cmd.LabelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")
	enterCmd.Flags().StringVar(&cmd.ImageSelector, "image-selector", "", "The image to search a pod for (e.g. nginx, nginx:latest, ${runtime.images.app}, nginx:${runtime.images.app.tag})")
	enterCmd.Flags().StringVar(&cmd.WorkingDirectory, "workdir", "", "The working directory where to open the terminal or execute the command")

	enterCmd.Flags().BoolVar(&cmd.Pick, "pick", true, "Select a pod / container if multiple are found")
	enterCmd.Flags().BoolVar(&cmd.Wait, "wait", false, "Wait for the pod(s) to start if they are not running")
	enterCmd.Flags().BoolVar(&cmd.Reconnect, "reconnect", false, "Will reconnect the terminal if an unexpected return code is encountered")

	return enterCmd
}

// Run executes the command logic
func (cmd *EnterCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	logger := f.GetLog()
	configOptions := cmd.ToConfigOptions(logger)
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

	// Execute plugin hook
	err = hook.ExecuteHooks(client, nil, nil, nil, nil, "enter")
	if err != nil {
		return err
	}

	// Build params
	selectorOptions := targetselector.NewOptionsFromFlags(cmd.Container, cmd.LabelSelector, cmd.Namespace, cmd.Pod, cmd.Pick)

	// get image selector if specified
	imageSelector, err := getImageSelector(client, configLoader, configOptions, cmd.Image, cmd.ImageSelector, logger)
	if err != nil {
		return err
	}

	// set image selector
	selectorOptions.ImageSelector = imageSelector

	// Start terminal
	stdout, stderr, stdin := defaultStdStreams(cmd.Stdout, cmd.Stderr, cmd.Stdin)
	exitCode, err := f.NewServicesClient(nil, nil, client, logger).StartTerminal(selectorOptions, args, cmd.WorkingDirectory, make(chan error), cmd.Wait, cmd.Reconnect, stdout, stderr, stdin)
	if err != nil {
		return err
	} else if exitCode != 0 {
		return &exit.ReturnCodeError{
			ExitCode: exitCode,
		}
	}

	return nil
}
