package cmd

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/resume"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

// AttachCmd is a struct that defines a command call for "enter"
type AttachCmd struct {
	*flags.GlobalFlags

	LabelSelector string
	Container     string
	Pod           string
	Pick          bool
}

// NewAttachCmd creates a new attach command
func NewAttachCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
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
		RunE: cmd.Run,
	}

	attachCmd.Flags().StringVarP(&cmd.Container, "container", "c", "", "Container name within pod where to execute command")
	attachCmd.Flags().StringVar(&cmd.Pod, "pod", "", "Pod to open a shell to")
	attachCmd.Flags().StringVarP(&cmd.LabelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")

	attachCmd.Flags().BoolVar(&cmd.Pick, "pick", false, "Select a pod")

	return attachCmd
}

// Run executes the command logic
func (cmd *AttachCmd) Run(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configLoader := loader.NewConfigLoader(cmd.ToConfigOptions(), log.GetInstance())
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}

	// Load generated config if possible
	var generatedConfig *generated.Config
	if configExists {
		generatedConfig, err = configLoader.Generated()
		if err != nil {
			return err
		}
	}

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, log.GetInstance())
	if err != nil {
		return err
	}

	// Get kubectl client
	client, err := kubectl.NewClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return errors.Wrap(err, "new kube client")
	}

	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, log.GetInstance())
	if err != nil {
		return err
	}

	// Signal that we are working on the space if there is any
	err = resume.NewSpaceResumer(client, log.GetInstance()).ResumeSpace(true)
	if err != nil {
		return err
	}

	// Build params
	selectorParameter := &targetselector.SelectorParameter{
		CmdParameter: targetselector.CmdParameter{
			ContainerName: cmd.Container,
			LabelSelector: cmd.LabelSelector,
			Namespace:     cmd.Namespace,
			PodName:       cmd.Pod,
		},
	}
	if cmd.Pick != false {
		selectorParameter.CmdParameter.Pick = &cmd.Pick
	}

	servicesClient := services.NewClient(nil, nil, client, selectorParameter, log.GetInstance())

	// Start attach
	return servicesClient.StartAttach(nil, make(chan error))
}
