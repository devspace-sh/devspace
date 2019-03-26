package cmd

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// EnterCmd is a struct that defines a command call for "enter"
type EnterCmd struct {
	flags *EnterCmdFlags
}

// EnterCmdFlags are the flags available for the enter-command
type EnterCmdFlags struct {
	selector      string
	namespace     string
	labelSelector string
	container     string
	pod           string
	switchContext bool
	pick          bool
}

func init() {
	cmd := &EnterCmd{
		flags: &EnterCmdFlags{},
	}

	cobraCmd := &cobra.Command{
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
	rootCmd.AddCommand(cobraCmd)

	cobraCmd.Flags().StringVarP(&cmd.flags.selector, "selector", "s", "", "Selector name (in config) to select pod/container for terminal")
	cobraCmd.Flags().StringVarP(&cmd.flags.container, "container", "c", "", "Container name within pod where to execute command")
	cobraCmd.Flags().StringVar(&cmd.flags.pod, "pod", "", "Pod to open a shell to")
	cobraCmd.Flags().StringVarP(&cmd.flags.labelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")
	cobraCmd.Flags().StringVarP(&cmd.flags.namespace, "namespace", "n", "", "Namespace where to select pods")
	cobraCmd.Flags().BoolVar(&cmd.flags.switchContext, "switch-context", false, "Switch kubectl context to the DevSpace context")
	cobraCmd.Flags().BoolVarP(&cmd.flags.pick, "pick", "p", false, "Select a pod to stream logs from")
}

// Run executes the command logic
func (cmd *EnterCmd) Run(cobraCmd *cobra.Command, args []string) {
	// Set config root
	_, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}

	// Get kubectl client
	kubectl, err := kubectl.NewClientWithContextSwitch(cmd.flags.switchContext)
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %v", err)
	}

	// Build params
	params := targetselector.CmdParameter{}
	if cmd.flags.selector != "" {
		params.Selector = &cmd.flags.selector
	}
	if cmd.flags.container != "" {
		params.ContainerName = &cmd.flags.container
	}
	if cmd.flags.labelSelector != "" {
		params.LabelSelector = &cmd.flags.labelSelector
	}
	if cmd.flags.namespace != "" {
		params.Namespace = &cmd.flags.namespace
	}
	if cmd.flags.pod != "" {
		params.PodName = &cmd.flags.pod
	}
	if cmd.flags.pick != false {
		params.Pick = &cmd.flags.pick
	}

	// Start terminal
	err = services.StartTerminal(kubectl, params, args, make(chan error), log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}
}
