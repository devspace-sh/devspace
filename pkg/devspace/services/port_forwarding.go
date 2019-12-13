package services

import (
	"strconv"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	logpkg "github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/devspace-cloud/devspace/pkg/util/port"
	"github.com/pkg/errors"
)

// StartPortForwarding starts the port forwarding functionality
func (serviceClient *client) StartPortForwarding() error {
	if serviceClient.config.Dev == nil {
		return nil
	}

	for _, portForwarding := range serviceClient.config.Dev.Ports {
		err := serviceClient.startForwarding(portForwarding, serviceClient.log)
		if err != nil {
			return err
		}
	}

	return nil
}

func (serviceClient *client) startForwarding(portForwarding *latest.PortForwardingConfig, log logpkg.Logger) error {
	var imageSelector []string
	if portForwarding.ImageName != "" && serviceClient.generated != nil {
		imageConfigCache := serviceClient.generated.GetActive().GetImageCache(portForwarding.ImageName)
		if imageConfigCache.ImageName != "" {
			imageSelector = []string{imageConfigCache.ImageName + ":" + imageConfigCache.Tag}
		}
	}

	selector, err := targetselector.NewTargetSelector(serviceClient.config, serviceClient.client, &targetselector.SelectorParameter{
		ConfigParameter: targetselector.ConfigParameter{
			Namespace:     portForwarding.Namespace,
			LabelSelector: portForwarding.LabelSelector,
		},
	}, false, imageSelector)
	if err != nil {
		return errors.Errorf("Error creating target selector: %v", err)
	}

	log.StartWait("Port-Forwarding: Waiting for containers to start...")
	pod, err := selector.GetPod(log)
	log.StopWait()
	if err != nil {
		return errors.Errorf("%s: %s", message.SelectorErrorPod, err.Error())
	} else if pod == nil {
		return nil
	}

	ports := make([]string, len(portForwarding.PortMappings))
	addresses := make([]string, len(portForwarding.PortMappings))

	for index, value := range portForwarding.PortMappings {
		if value.LocalPort == nil {
			return errors.Errorf("port is not defined in portmapping %d", index)
		}

		localPort := strconv.Itoa(*value.LocalPort)
		remotePort := localPort
		if value.RemotePort != nil {
			remotePort = strconv.Itoa(*value.RemotePort)
		}

		open, _ := port.Check(*value.LocalPort)
		if open == false {
			serviceClient.log.Warnf("Seems like port %d is already in use. Is another application using that port?", *value.LocalPort)
		}

		ports[index] = localPort + ":" + remotePort
		if value.BindAddress == "" {
			addresses[index] = "localhost"
		} else {
			addresses[index] = value.BindAddress
		}
	}

	readyChan := make(chan struct{})
	errorChan := make(chan error)

	pf, err := serviceClient.client.NewPortForwarder(pod, ports, addresses, make(chan struct{}), readyChan, errorChan)
	if err != nil {
		return errors.Errorf("Error starting port forwarding: %v", err)
	}

	go func() {
		err := pf.ForwardPorts()
		if err != nil {
			serviceClient.log.Fatalf("Error forwarding ports: %v", err)
		}
	}()

	// Wait till forwarding is ready
	select {
	case <-readyChan:
		log.Donef("Port forwarding started on %s", strings.Join(ports, ", "))
	case <-time.After(20 * time.Second):
		return errors.Errorf("Timeout waiting for port forwarding to start")
	}

	go func(portForwarding *latest.PortForwardingConfig) {
		err := <-errorChan
		if err != nil {
			pf.Close()
			for {
				err = serviceClient.startForwarding(portForwarding, logpkg.Discard)
				if err != nil {
					serviceClient.log.Errorf("Error restarting port-forwarding: %v", err)
					serviceClient.log.Errorf("Will try again in 3 seconds", err)
					continue
				}

				time.Sleep(time.Second * 3)
				break
			}
		}
	}(portForwarding)

	return nil
}
