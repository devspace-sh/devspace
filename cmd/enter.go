package cmd

import (
	"os"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/analytics/cloudanalytics"
	"github.com/devspace-cloud/devspace/pkg/util/log"

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

// NewEnterCmd creates a new init command
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
devspace enter -s my-selector
devspace enter -c my-container
devspace enter bash -n my-namespace
devspace enter bash -l release=test
#######################################################`,
		Run: cmd.Run,
	}

	enterCmd.Flags().StringVarP(&cmd.Container, "container", "c", "", "Container name within pod where to execute command")
	enterCmd.Flags().StringVar(&cmd.Pod, "pod", "", "Pod to open a shell to")
	enterCmd.Flags().StringVarP(&cmd.LabelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")

	enterCmd.Flags().BoolVar(&cmd.Pick, "pick", false, "Select a pod")

	return enterCmd
}

// Run executes the command logic
func (cmd *EnterCmd) Run(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}

	// Load generated config if possible
	var generatedConfig *generated.Config
	if configExists {
		generatedConfig, err = generated.LoadConfig(cmd.Profile)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Get kubectl client
	client, err := kubectl.NewClientFromContext(cmd.KubeContext, cmd.Namespace, false)
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %v", err)
	}

	err = client.PrintWarning(generatedConfig, false, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	// Signal that we are working on the space if there is any
	err = cloud.ResumeSpace(client, true, log.GetInstance())
	if err != nil {
		log.Fatal(err)
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
	exitCode, err := services.StartTerminal(nil, client, selectorParameter, args, nil, make(chan error), log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	cloudanalytics.SendCommandEvent(nil)
	os.Exit(exitCode)
}
