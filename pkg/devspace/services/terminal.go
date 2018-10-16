package services

import (
	"strings"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/log"
	"k8s.io/client-go/kubernetes"
	kubectlExec "k8s.io/client-go/util/exec"
)

// StartTerminal opens a new terminal
func StartTerminal(client *kubernetes.Clientset, containerNameOverride string, args []string, log log.Logger) {
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
	labelSelector := "release=" + GetNameOfFirstHelmDeployment()
	if config.DevSpace.Terminal != nil && config.DevSpace.Terminal.LabelSelector != nil {
		labels := make([]string, 0, len(*config.DevSpace.Terminal.LabelSelector))
		for key, value := range *config.DevSpace.Terminal.LabelSelector {
			labels = append(labels, key+"="+*value)
		}

		labelSelector = strings.Join(labels, ", ")
	}

	// Get first running pod
	log.StartWait("Waiting for pods to become running")
	pod, err := kubectl.GetNewestRunningPod(client, labelSelector, namespace)
	log.StopWait()
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

// GetNameOfFirstHelmDeployment retrieves the first helm deployment name
func GetNameOfFirstHelmDeployment() string {
	config := configutil.GetConfig()

	if config.DevSpace.Deployments != nil {
		for _, deploymentConfig := range *config.DevSpace.Deployments {
			if deploymentConfig.Helm != nil {
				return *deploymentConfig.Name
			}
		}
	}

	return configutil.DefaultDevspaceDeploymentName
}
