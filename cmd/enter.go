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
	container     string
	namespace     string
	labelSelector string
}

func init() {
	cmd := &EnterCmd{
		flags: &EnterCmdFlags{},
	}

	cobraCmd := &cobra.Command{
		Use:   "enter",
		Short: "Enter your DevSpace",
		Long: `
#######################################################
################## devspace enter #####################
#######################################################
Execute a command or start a new terminal in your 
devspace:

devspace enter
devspace enter bash
devspace enter -c myContainer
devspace enter bash -n my-namespace
devspace enter bash -l release=test
#######################################################`,
		Run: cmd.Run,
	}
	rootCmd.AddCommand(cobraCmd)

	cobraCmd.Flags().StringVarP(&cmd.flags.container, "container", "c", "", "Container name within pod where to execute command")
	cobraCmd.Flags().StringVarP(&cmd.flags.namespace, "namespace", "n", "", "Namespace where to select pods")
	cobraCmd.Flags().StringVarP(&cmd.flags.labelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")
}

// Run executes the command logic
func (cmd *EnterCmd) Run(cobraCmd *cobra.Command, args []string) {
	var err error
	log.StartFileLogging()

	cmd.kubectl, err = kubectl.NewClient()
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %v", err)
	}

	services.StartTerminal(cmd.kubectl, cmd.flags.container, cmd.flags.labelSelector, cmd.flags.namespace, args, log.GetInstance())
}
