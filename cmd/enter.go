package cmd

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/exit"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

// EnterCmd is a struct that defines a command call for "enter"
type EnterCmd struct {
	*flags.GlobalFlags

	LabelSelector string
	Container     string
	Pod           string
	Pick          bool
}

// NewEnterCmd creates a new enter command
func NewEnterCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
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
		RunE: cmd.Run,
	}

	enterCmd.Flags().StringVarP(&cmd.Container, "container", "c", "", "Container name within pod where to execute command")
	enterCmd.Flags().StringVar(&cmd.Pod, "pod", "", "Pod to open a shell to")
	enterCmd.Flags().StringVarP(&cmd.LabelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")

	enterCmd.Flags().BoolVar(&cmd.Pick, "pick", false, "Select a pod")

	return enterCmd
}

// Run executes the command logic
func (cmd *EnterCmd) Run(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot(log.GetInstance())
	if err != nil {
		return err
	}

	// Load generated config if possible
	var generatedConfig *generated.Config
	if configExists {
		generatedConfig, err = generated.LoadConfig(cmd.Profile)
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
	err = cloud.ResumeSpace(client, true, log.GetInstance())
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

	// Start terminal
	exitCode, err := services.StartTerminal(nil, client, selectorParameter, args, nil, make(chan error), false, log.GetInstance())
	if err != nil {
		return err
	} else if exitCode != 0 {
		return &exit.ReturnCodeError{
			ExitCode: exitCode,
		}
	}

	return nil
}
