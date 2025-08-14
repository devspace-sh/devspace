package cmd

import (
	"context"
	"github.com/loft-sh/devspace/cmd/flags"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/services/attach"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

// AttachCmd is a struct that defines a command call for "enter"
type AttachCmd struct {
	*flags.GlobalFlags

	LabelSelector string
	ImageSelector string
	Container     string
	Pod           string
	Pick          bool
}

// NewAttachCmd creates a new attach command
func NewAttachCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
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
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f, cobraCmd, args)
		},
	}

	attachCmd.Flags().StringVarP(&cmd.Container, "container", "c", "", "Container name within pod where to execute command")
	attachCmd.Flags().StringVar(&cmd.Pod, "pod", "", "Pod to open a shell to")
	attachCmd.Flags().StringVar(&cmd.ImageSelector, "image-selector", "", "The image to search a pod for (e.g. nginx, nginx:latest, ${runtime.images.app}, nginx:${runtime.images.app.tag})")
	attachCmd.Flags().StringVarP(&cmd.LabelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")

	attachCmd.Flags().BoolVar(&cmd.Pick, "pick", true, "Select a pod")

	return attachCmd
}

// Run executes the command logic
func (cmd *AttachCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	log := f.GetLog()
	configOptions := cmd.ToConfigOptions()
	configLoader, err := f.NewConfigLoader(cmd.ConfigPath)
	if err != nil {
		return err
	}
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return err
	}

	// Get kubectl client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace)
	if err != nil {
		return errors.Wrap(err, "new kube client")
	}

	// Load generated config if possible
	if configExists {
		localCache, err := configLoader.LoadLocalCache()
		if err != nil {
			return err
		}

		// If the current kube context or namespace is different from old,
		// show warnings and reset kube client if necessary
		client, err = kubectl.CheckKubeContext(client, localCache, cmd.NoWarn, cmd.SwitchContext, false, log)
		if err != nil {
			return err
		}
	}

	// create the context
	ctx := devspacecontext.NewContext(context.Background(), nil, log).WithKubeClient(client)

	// Execute plugin hook
	err = hook.ExecuteHooks(ctx, nil, "attach")
	if err != nil {
		return err
	}

	// get image selector if specified
	imageSelector, err := getImageSelector(ctx, configLoader, configOptions, cmd.ImageSelector)
	if err != nil {
		return err
	}

	// Build params
	options := targetselector.NewOptionsFromFlags(cmd.Container, cmd.LabelSelector, imageSelector, cmd.Namespace, cmd.Pod).
		WithPick(cmd.Pick).
		WithWait(false).
		WithQuestion("Which pod do you want to attach to?")

	// Start attach
	return attach.StartAttachFromCMD(ctx, targetselector.NewTargetSelector(options))
}
