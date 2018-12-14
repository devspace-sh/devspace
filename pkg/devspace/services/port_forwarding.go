package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/portforward"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/log"
)

// StartPortForwarding starts the port forwarding functionality
func StartPortForwarding(client *kubernetes.Clientset, log log.Logger) ([]*portforward.PortForwarder, error) {
	config := configutil.GetConfig()
	if config.DevSpace.Ports != nil {
		portforwarder := make([]*portforward.PortForwarder, 0, len(*config.DevSpace.Ports))

		for _, portForwarding := range *config.DevSpace.Ports {
			if portForwarding.ResourceType == nil || *portForwarding.ResourceType == "pod" {
				var labelSelector map[string]*string
				namespace := ""

				if portForwarding.Service != nil {
					service, err := configutil.GetService(*portForwarding.Service)
					if err != nil {
						log.Fatalf("Error resolving service name: %v", err)
					}

					labelSelector = *service.LabelSelector
					if service.Namespace != nil && *service.Namespace != "" {
						namespace = *service.Namespace
					}
				} else {
					labelSelector = *portForwarding.LabelSelector
					if portForwarding.Namespace != nil && *portForwarding.Namespace != "" {
						namespace = *portForwarding.Namespace
					}
				}

				labels := make([]string, 0, len(labelSelector)-1)
				for key, value := range labelSelector {
					labels = append(labels, key+"="+*value)
				}

				log.StartWait("Waiting for pods to become running")
				pod, err := kubectl.GetNewestRunningPod(client, strings.Join(labels, ", "), namespace, time.Second*120)
				log.StopWait()

				if err != nil {
					return nil, fmt.Errorf("Error starting port-forwarding: Unable to list devspace pods: %s", err.Error())
				} else if pod != nil {
					ports := make([]string, len(*portForwarding.PortMappings))
					addresses := make([]string, len(*portForwarding.PortMappings))

					for index, value := range *portForwarding.PortMappings {
						ports[index] = strconv.Itoa(*value.LocalPort) + ":" + strconv.Itoa(*value.RemotePort)
						if value.BindAddress == nil {
							addresses[index] = "127.0.0.1"
						} else {
							addresses[index] = *value.BindAddress
						}
					}

					readyChan := make(chan struct{})

					pf, err := kubectl.NewPortForwarder(client, pod, ports, addresses, make(chan struct{}), readyChan)
					if err != nil {
						log.Fatalf("Error starting port forwarding: %v", err)
					}

					go func() {
						err := pf.ForwardPorts()
						if err != nil {
							log.Errorf("Error forwarding ports: %v", err)
						}
					}()

					// Wait till forwarding is ready
					select {
					case <-readyChan:
						log.Donef("Port forwarding started on %s", strings.Join(ports, ", "))

						portforwarder = append(portforwarder, pf)
					case <-time.After(20 * time.Second):
						return nil, fmt.Errorf("Timeout waiting for port forwarding to start")
					}
				}
			} else {
				log.Warn("Currently only pod resource type is supported for portforwarding")
			}
		}

		return portforwarder, nil
	}

	return nil, nil
}
