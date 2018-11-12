package cmd

import (
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/devspace/services"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

// EnterCmd is a struct that defines a command call for "enter"
type EnterCmd struct {
	flags   *EnterCmdFlags
	kubectl *kubernetes.Clientset
}

// EnterCmdFlags are the flags available for the enter-command
type EnterCmdFlags struct {
	service       string
	namespace     string
	labelSelector string
	container     string
	switchContext bool
}

func init() {
	cmd := &EnterCmd{
		flags: &EnterCmdFlags{},
	}

	cobraCmd := &cobra.Command{
		Use:   "enter",
		Short: "Start a new terminal session",
		Long: `
#######################################################
################## devspace enter #####################
#######################################################
Execute a command or start a new terminal in your 
devspace:

devspace enter
devspace enter bash
devspace enter -s my-service
devspace enter -c my-container
devspace enter bash -n my-namespace
devspace enter bash -l release=test
#######################################################`,
		Run: cmd.Run,
	}
	rootCmd.AddCommand(cobraCmd)

	cobraCmd.Flags().StringVarP(&cmd.flags.service, "service", "s", "", "Service name (in config) to select pod/container for terminal")
	cobraCmd.Flags().StringVarP(&cmd.flags.container, "container", "c", "", "Container name within pod where to execute command")
	cobraCmd.Flags().StringVarP(&cmd.flags.labelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")
	cobraCmd.Flags().StringVarP(&cmd.flags.namespace, "namespace", "n", "", "Namespace where to select pods")
	cobraCmd.Flags().BoolVar(&cmd.flags.switchContext, "switch-context", false, "Switch kubectl context to the devspace context")
}

// Run executes the command logic
func (cmd *EnterCmd) Run(cobraCmd *cobra.Command, args []string) {
	var err error
	log.StartFileLogging()

	cmd.kubectl, err = kubectl.NewClientWithContextSwitch(cmd.flags.switchContext)
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %v", err)
	}

	err = services.StartTerminal(cmd.kubectl, cmd.flags.service, cmd.flags.container, cmd.flags.labelSelector, cmd.flags.namespace, args, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}
}
