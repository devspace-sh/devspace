package cmd

import (
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	helmClient "github.com/covexo/devspace/pkg/devspace/deploy/helm"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	kubectlExec "k8s.io/client-go/util/exec"
)

// EnterCmd is a struct that defines a command call for "enter"
type EnterCmd struct {
	flags   *EnterCmdFlags
	helm    *helmClient.HelmClientWrapper
	kubectl *kubernetes.Clientset
	pod     *k8sv1.Pod
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

	log.StartWait("Initializing helm client")
	cmd.helm, err = helmClient.NewClient(cmd.kubectl, false)
	log.StopWait()
	if err != nil {
		log.Fatalf("Error initializing helm client: %s", err.Error())
	}

	// Check if we find a running release pod
	log.StartWait("Find a running devspace pod")
	pod, err := getRunningDevSpacePod(cmd.helm, cmd.kubectl)
	log.StopWait()
	if err != nil {
		log.Fatal("Cannot find a running devspace pod")
	}

	enterTerminal(cmd.kubectl, pod, cmd.flags.container, args)
}

func enterTerminal(client *kubernetes.Clientset, pod *k8sv1.Pod, containerNameOverride string, args []string) {
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
