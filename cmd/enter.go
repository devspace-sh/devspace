package cmd

import (
	"strings"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	kubectlExec "k8s.io/client-go/util/exec"
)

// EnterCmd is a struct that defines a command call for "enter"
type EnterCmd struct {
	flags   *EnterCmdFlags
	kubectl *kubernetes.Clientset
}

// EnterCmdFlags are the flags available for the enter-command
type EnterCmdFlags struct {
	container string
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
#######################################################`,
		Run: cmd.Run,
	}
	rootCmd.AddCommand(cobraCmd)

	cobraCmd.Flags().StringVarP(&cmd.flags.container, "container", "c", "", "Container name within pod where to execute command")
}

// Run executes the command logic
func (cmd *EnterCmd) Run(cobraCmd *cobra.Command, args []string) {
	var err error
	log.StartFileLogging()

	cmd.kubectl, err = kubectl.NewClient()
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %v", err)
	}

	enterTerminal(cmd.kubectl, cmd.flags.container, args)
}

func enterTerminal(client *kubernetes.Clientset, containerNameOverride string, args []string) {
	var command []string
	config := configutil.GetConfig()

	if len(args) == 0 && (config.DevSpace.Terminal.Command == nil || len(*config.DevSpace.Terminal.Command) == 0) {
		command = []string{
			"sh",
			"-c",
			"command -v bash >/dev/null 2>&1 && exec bash || exec sh",
		}
	} else {
		if len(args) > 0 {
			command = args
		} else {
			for _, cmd := range *config.DevSpace.Terminal.Command {
				command = append(command, *cmd)
			}
		}
	}

	// Select pods
	namespace := ""
	if config.DevSpace.Terminal != nil && config.DevSpace.Terminal.Namespace != nil {
		namespace = *config.DevSpace.Terminal.Namespace
	}

	// Retrieve pod from label selector
	labelSelector := "release=" + getNameOfFirstHelmDeployment()
	if config.DevSpace.Terminal != nil && config.DevSpace.Terminal.LabelSelector != nil {
		labels := make([]string, 0, len(*config.DevSpace.Terminal.LabelSelector))
		for key, value := range *config.DevSpace.Terminal.LabelSelector {
			labels = append(labels, key+"="+*value)
		}

		labelSelector = strings.Join(labels, ", ")
	}

	// Get first running pod
	pod, err := kubectl.GetFirstRunningPod(client, labelSelector, namespace)
	if err != nil {
		log.Fatalf("Cannot find running pod: %v", err)
	}

	// Get container name
	containerName := pod.Spec.Containers[0].Name
	if containerNameOverride != "" {
		containerName = containerNameOverride
	} else if config.DevSpace.Terminal.ContainerName != nil {
		containerName = *config.DevSpace.Terminal.ContainerName
	}

	_, _, _, terminalErr := kubectl.Exec(client, pod, containerName, command, true, nil)
	if terminalErr != nil {
		if _, ok := terminalErr.(kubectlExec.CodeExitError); ok == false {
			log.Fatalf("Unable to start terminal session: %v", terminalErr)
		}
	}
}
