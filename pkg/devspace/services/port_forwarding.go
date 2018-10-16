package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/log"
)

// StartPortForwarding starts the port forwarding functionality
func StartPortForwarding(client *kubernetes.Clientset, log log.Logger) error {
	config := configutil.GetConfig()

	for _, portForwarding := range *config.DevSpace.PortForwarding {
		if portForwarding.ResourceType == nil || *portForwarding.ResourceType == "pod" {
			if len(*portForwarding.LabelSelector) > 0 {
				labels := make([]string, 0, len(*portForwarding.LabelSelector))
				for key, value := range *portForwarding.LabelSelector {
					labels = append(labels, key+"="+*value)
				}

				namespace := ""
				if portForwarding.Namespace != nil && *portForwarding.Namespace != "" {
					namespace = *portForwarding.Namespace
				}

				pod, err := kubectl.GetNewestRunningPod(client, strings.Join(labels, ", "), namespace)
				if err != nil {
					return fmt.Errorf("Unable to list devspace pods: %s", err.Error())
				} else if pod != nil {
					ports := make([]string, len(*portForwarding.PortMappings))

					for index, value := range *portForwarding.PortMappings {
						ports[index] = strconv.Itoa(*value.LocalPort) + ":" + strconv.Itoa(*value.RemotePort)
					}

					readyChan := make(chan struct{})

					go kubectl.ForwardPorts(client, pod, ports, make(chan struct{}), readyChan)

					// Wait till forwarding is ready
					select {
					case <-readyChan:
						log.Donef("Port forwarding started on %s", strings.Join(ports, ", "))
					case <-time.After(5 * time.Second):
						return fmt.Errorf("Timeout waiting for port forwarding to start")
					}
				}
			}
		} else {
			log.Warn("Currently only pod resource type is supported for portforwarding")
		}
	}

	return nil
}
