package cmd

import (
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	latest "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/analytics/cloudanalytics"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/spf13/cobra"
)

// EnterCmd is a struct that defines a command call for "enter"
type EnterCmd struct {
	Selector      string
	Namespace     string
	LabelSelector string
	Container     string
	Pod           string
	SwitchContext bool
	Pick          bool
}

// NewEnterCmd creates a new init command
func NewEnterCmd() *cobra.Command {
	cmd := &EnterCmd{}

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
devspace enter -p # Select pod to enter
devspace enter bash
devspace enter -s my-selector
devspace enter -c my-container
devspace enter bash -n my-namespace
devspace enter bash -l release=test
#######################################################`,
		Run: cmd.Run,
	}

	enterCmd.Flags().StringVarP(&cmd.Selector, "selector", "s", "", "Selector name (in config) to select pod/container for terminal")
	enterCmd.Flags().StringVarP(&cmd.Container, "container", "c", "", "Container name within pod where to execute command")
	enterCmd.Flags().StringVar(&cmd.Pod, "pod", "", "Pod to open a shell to")
	enterCmd.Flags().StringVarP(&cmd.LabelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")
	enterCmd.Flags().StringVarP(&cmd.Namespace, "namespace", "n", "", "Namespace where to select pods")
	enterCmd.Flags().BoolVar(&cmd.SwitchContext, "switch-context", false, "Switch kubectl context to the DevSpace context")
	enterCmd.Flags().BoolVarP(&cmd.Pick, "pick", "p", false, "Select a pod")

	return enterCmd
}

// Run executes the command logic
func (cmd *EnterCmd) Run(cobraCmd *cobra.Command, args []string) {
	// Set config root
	_, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}

	// Get config
	var config *latest.Config
	if configutil.ConfigExists() {
		config = configutil.GetConfig()

		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			log.Fatal(err)
		}

		err = cloud.ResumeSpace(config, generatedConfig, true, log.GetInstance())
		if err != nil {
			log.Fatal(err)
		}
	}

	// Get kubectl client
	kubectl, err := kubectl.NewClientWithContextSwitch(config, cmd.SwitchContext)
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %v", err)
	}

	// Build params
	params := targetselector.CmdParameter{}
	if cmd.Selector != "" {
		params.Selector = &cmd.Selector
	}
	if cmd.Container != "" {
		params.ContainerName = &cmd.Container
	}
	if cmd.LabelSelector != "" {
		params.LabelSelector = &cmd.LabelSelector
	}
	if cmd.Namespace != "" {
		params.Namespace = &cmd.Namespace
	}
	if cmd.Pod != "" {
		params.PodName = &cmd.Pod
	}
	if cmd.Pick != false {
		params.Pick = &cmd.Pick
	}

	// Start terminal
	exitCode, err := services.StartTerminal(config, kubectl, params, args, make(chan error), log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	cloudanalytics.SendCommandEvent(nil)
	os.Exit(exitCode)
}
