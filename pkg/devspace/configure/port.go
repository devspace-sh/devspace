package configure

import (
	"strconv"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/pkg/errors"
)

// GetNameOfFirstDeployment retrieves the first deployment name
func GetNameOfFirstDeployment(config *latest.Config) string {
	if config.Deployments != nil {
		for _, deploymentConfig := range config.Deployments {
			if deploymentConfig.Helm != nil && deploymentConfig.Helm.ComponentChart != nil && *deploymentConfig.Helm.ComponentChart == true {
				return deploymentConfig.Name
			}
		}
	}

	return "devspace"
}

// AddPort adds a port to the config
func AddPort(baseConfig *latest.Config, namespace, labelSelector string, args []string) error {
	var labelSelectorMap map[string]string
	var err error

	portMappings, err := parsePortMappings(args[0])
	if err != nil {
		return errors.Errorf("Error parsing port mappings: %s", err.Error())
	}

	// Add to first existing port mapping if labelselector and service name are empty
	if labelSelector == "" && baseConfig.Dev != nil && baseConfig.Dev.Ports != nil && len(baseConfig.Dev.Ports) > 0 {
		if (baseConfig.Dev.Ports)[0].PortMappings == nil {
			(baseConfig.Dev.Ports)[0].PortMappings = []*latest.PortMapping{}
		}

		for _, portMapping := range portMappings {
			(baseConfig.Dev.Ports)[0].PortMappings = append((baseConfig.Dev.Ports)[0].PortMappings, portMapping)
		}

		return nil
	} else if labelSelector == "" {
		labelSelector = "app.kubernetes.io/component=" + GetNameOfFirstDeployment(baseConfig)
	}

	if labelSelectorMap == nil {
		labelSelectorMap, err = parseSelectors(labelSelector)
		if err != nil {
			return errors.Errorf("Error parsing selectors: %s", err.Error())
		}
	}

	insertOrReplacePortMapping(baseConfig, namespace, labelSelectorMap, portMappings)
	return nil
}

// RemovePort removes a port from the config
func RemovePort(baseConfig *latest.Config, removeAll bool, labelSelector string, args []string) error {
	labelSelectorMap, err := parseSelectors(labelSelector)
	if err != nil {
		return errors.Errorf("Error parsing selectors: %s", err.Error())
	}

	argPorts := ""
	if len(args) == 1 {
		argPorts = args[0]
	}

	if len(labelSelectorMap) == 0 && removeAll == false && argPorts == "" {
		return errors.Errorf("You have to specify at least one of the supported flags")
	}

	ports := strings.Split(argPorts, ",")
	if baseConfig.Dev.Ports != nil && len(baseConfig.Dev.Ports) > 0 {
		newPortForwards := make([]*latest.PortForwardingConfig, 0, len(baseConfig.Dev.Ports)-1)

		for _, v := range baseConfig.Dev.Ports {
			if removeAll {
				continue
			}

			newPortMappings := []*latest.PortMapping{}
			for _, pm := range v.PortMappings {
				if pm.LocalPort != nil && containsPort(strconv.Itoa(*pm.LocalPort), ports) {
					continue
				}
				if pm.RemotePort != nil && containsPort(strconv.Itoa(*pm.RemotePort), ports) {
					continue
				}

				newPortMappings = append(newPortMappings, pm)
			}

			if len(newPortMappings) > 0 {
				v.PortMappings = newPortMappings
				newPortForwards = append(newPortForwards, v)
			}
		}

		baseConfig.Dev.Ports = newPortForwards
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

func insertOrReplacePortMapping(config *latest.Config, namespace string, labelSelectorMap map[string]string, portMappings []*latest.PortMapping) {
	if config.Dev.Ports == nil {
		config.Dev.Ports = []*latest.PortForwardingConfig{}
	}

	// Check if we should add to existing port mapping
	for _, v := range config.Dev.Ports {
		var selectors map[string]string

		if v.LabelSelector != nil {
			selectors = v.LabelSelector
		} else {
			selectors = map[string]string{}
		}

		if areLabelMapsEqual(selectors, labelSelectorMap) {
			portMap := append(v.PortMappings, portMappings...)
			v.PortMappings = portMap
			return
		}
	}

	newPortConfig := &latest.PortForwardingConfig{
		PortMappings: portMappings,
	}
	if labelSelectorMap != nil {
		newPortConfig.LabelSelector = labelSelectorMap
	}
	if namespace != "" {
		newPortConfig.Namespace = namespace
	}

	portMap := append(config.Dev.Ports, newPortConfig)
	config.Dev.Ports = portMap
}

func parsePortMappings(portMappingsString string) ([]*latest.PortMapping, error) {
	portMappings := make([]*latest.PortMapping, 0, 1)
	portMappingsSplitted := strings.Split(portMappingsString, ",")

	for _, v := range portMappingsSplitted {
		portMapping := strings.Split(v, ":")

		if len(portMapping) != 1 && len(portMapping) != 2 {
			return nil, errors.Errorf("Error parsing port mapping: %s", v)
		}

		portMappingStruct := &latest.PortMapping{}
		firstPort, err := strconv.Atoi(portMapping[0])

		if err != nil {
			return nil, err
		}

		if len(portMapping) == 1 {
			portMappingStruct.LocalPort = &firstPort
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
