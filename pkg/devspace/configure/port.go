package configure

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/devspace/services"
)

// AddPort adds a port to the config
func AddPort(namespace, labelSelector, serviceName string, args []string) error {

	var labelSelectorMap map[string]*string
	var err error

	if labelSelector != "" && serviceName != "" {
		return fmt.Errorf("both service and label-selector specified. This is illegal because the label-selector is already specified in the referenced service. Therefore defining both is redundant")
	}

	if labelSelector == "" {
		config := configutil.GetConfig()

		if config.DevSpace != nil && config.DevSpace.Services != nil && len(*config.DevSpace.Services) > 0 {
			services := *config.DevSpace.Services

			var service *v1.ServiceConfig
			if serviceName != "" {
				service = getServiceWithName(*config.DevSpace.Services, serviceName)
				if service == nil {
					return fmt.Errorf("no service with name %v exists", serviceName)
				}
			} else {
				service = services[0]
			}
			labelSelectorMap = *service.LabelSelector
		} else {
			labelSelector = "release=" + services.GetNameOfFirstHelmDeployment()
		}
	}

	if labelSelectorMap == nil {
		labelSelectorMap, err = parseSelectors(labelSelector)
		if err != nil {
			return fmt.Errorf("Error parsing selectors: %s", err.Error())
		}
	}

	portMappings, err := parsePortMappings(args[0])
	if err != nil {
		return fmt.Errorf("Error parsing port mappings: %s", err.Error())
	}

	insertOrReplacePortMapping(namespace, labelSelectorMap, serviceName, portMappings)

	err = configutil.SaveConfig()
	if err != nil {
		return fmt.Errorf("Couldn't save config file: %s", err.Error())
	}

	return nil
}

// RemovePort removes a port from the config
func RemovePort(removeAll bool, labelSelector string, args []string) error {
	config := configutil.GetConfig()

	labelSelectorMap, err := parseSelectors(labelSelector)
	if err != nil {
		return fmt.Errorf("Error parsing selectors: %s", err.Error())
	}

	argPorts := ""
	if len(args) == 1 {
		argPorts = args[0]
	}

	if len(labelSelectorMap) == 0 && removeAll == false && argPorts == "" {
		return fmt.Errorf("You have to specify at least one of the supported flags")
	}

	ports := strings.Split(argPorts, ",")

	if config.DevSpace.Ports != nil && len(*config.DevSpace.Ports) > 0 {
		newPortForwards := make([]*v1.PortForwardingConfig, 0, len(*config.DevSpace.Ports)-1)

		for _, v := range *config.DevSpace.Ports {
			if removeAll {
				continue
			}

			newPortMappings := []*v1.PortMapping{}

			for _, pm := range *v.PortMappings {
				if containsPort(strconv.Itoa(*pm.LocalPort), ports) || containsPort(strconv.Itoa(*pm.RemotePort), ports) {
					continue
				}

				newPortMappings = append(newPortMappings, pm)
			}

			if len(newPortMappings) > 0 {
				v.PortMappings = &newPortMappings
				newPortForwards = append(newPortForwards, v)
			}
		}

		config.DevSpace.Ports = &newPortForwards

		err = configutil.SaveConfig()
		if err != nil {
			return fmt.Errorf("Couldn't save config file: %s", err.Error())
		}
	}

	return nil
}

func containsPort(port string, ports []string) bool {
	for _, v := range ports {
		if strings.TrimSpace(v) == port {
			return true
		}
	}

	return false
}

func insertOrReplacePortMapping(namespace string, labelSelectorMap map[string]*string, serviceName string, portMappings []*v1.PortMapping) {
	config := configutil.GetConfig()

	if config.DevSpace.Ports == nil {
		config.DevSpace.Ports = &[]*v1.PortForwardingConfig{}
	}

	// Check if we should add to existing port mapping
	for _, v := range *config.DevSpace.Ports {
		var selectors map[string]*string

		if v.LabelSelector != nil {
			selectors = *v.LabelSelector
		} else {
			selectors = map[string]*string{}
		}

		if areLabelMapsEqual(selectors, labelSelectorMap) {
			portMap := append(*v.PortMappings, portMappings...)

			v.PortMappings = &portMap

			return
		}
	}

	//We set labelSelectorMap to nil since labelSelectorMap is already specified in service. Avoid redundance.
	if serviceName != "" {
		labelSelectorMap = nil
	}

	portMap := append(*config.DevSpace.Ports, &v1.PortForwardingConfig{
		ResourceType:  nil,
		LabelSelector: &labelSelectorMap,
		PortMappings:  &portMappings,
		Namespace:     &namespace,
		Service:       &serviceName,
	})

	config.DevSpace.Ports = &portMap
}

func parsePortMappings(portMappingsString string) ([]*v1.PortMapping, error) {
	portMappings := make([]*v1.PortMapping, 0, 1)
	portMappingsSplitted := strings.Split(portMappingsString, ",")

	for _, v := range portMappingsSplitted {
		portMapping := strings.Split(v, ":")

		if len(portMapping) != 1 && len(portMapping) != 2 {
			return nil, fmt.Errorf("Error parsing port mapping: %s", v)
		}

		portMappingStruct := &v1.PortMapping{}
		firstPort, err := strconv.Atoi(portMapping[0])

		if err != nil {
			return nil, err
		}

		if len(portMapping) == 1 {
			portMappingStruct.LocalPort = &firstPort

			portMappingStruct.RemotePort = portMappingStruct.LocalPort
		} else {
			portMappingStruct.LocalPort = &firstPort

			secondPort, err := strconv.Atoi(portMapping[1])

			if err != nil {
				return nil, err
			}
			portMappingStruct.RemotePort = &secondPort
		}

		portMappings = append(portMappings, portMappingStruct)
	}

	return portMappings, nil
}

func getServiceWithName(services []*v1.ServiceConfig, name string) *v1.ServiceConfig {

	for _, service := range services {
		if *service.Name == name {
			return service
		}
	}

	return nil

}
