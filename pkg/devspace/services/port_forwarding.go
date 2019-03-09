package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/portforward"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// StartPortForwarding starts the port forwarding functionality
func StartPortForwarding(client *kubernetes.Clientset, log log.Logger) ([]*portforward.PortForwarder, error) {
	config := configutil.GetConfig()
	if config.Dev.Ports != nil {
		portforwarder := make([]*portforward.PortForwarder, 0, len(*config.Dev.Ports))

		for _, portForwarding := range *config.Dev.Ports {
			selector, err := targetselector.NewTargetSelector(&targetselector.SelectorParameter{
				ConfigParameter: targetselector.ConfigParameter{
					Selector:      portForwarding.Selector,
					Namespace:     portForwarding.Namespace,
					LabelSelector: portForwarding.LabelSelector,
				},
			}, false)
			if err != nil {
				return nil, fmt.Errorf("Error creating target selector: %v", err)
			}

			log.StartWait("Port-Forwarding: Waiting for pods...")
			pod, err := selector.GetPod(client)
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
		}

		return portforwarder, nil
	}

	return nil, nil
}
