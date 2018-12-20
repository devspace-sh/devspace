package configure

import (
	"fmt"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/devspace/services"
)

//AddService adds an image to the devspace
func AddService(name string, labelSelector string, namespace string) error {
	config := configutil.GetConfig()

	var labelSelectorMap map[string]*string
	var err error

	if labelSelector == "" {
		config := configutil.GetConfig()

		if config.DevSpace != nil && config.DevSpace.Services != nil && len(*config.DevSpace.Services) > 0 {
			services := *config.DevSpace.Services
			labelSelectorMap = *services[0].LabelSelector
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

	if *config.DevSpace.Services == nil {
		emptyServiceList := make([]*v1.ServiceConfig, 0)
		config.DevSpace.Services = &emptyServiceList
	}

	servicesConfig := append(*config.DevSpace.Services, &v1.ServiceConfig{
		LabelSelector: &labelSelectorMap,
		Namespace:     &namespace,
		Name:          &name,
	})

	config.DevSpace.Services = &servicesConfig

	err = configutil.SaveConfig()
	if err != nil {
		return fmt.Errorf("Couldn't save config file: %s", err.Error())
	}

	return nil
}

//RemoveService removes a service from the devspace
func RemoveService(removeAll bool, name string, labelSelector string, namespace string) error {
	config := configutil.GetConfig()
	labelSelectorMap, err := parseSelectors(labelSelector)

	if err != nil {
		return fmt.Errorf("Error parsing selectors: %v", err)
	}

	if len(labelSelectorMap) == 0 && removeAll == false && name == "" && namespace == "" {
		return fmt.Errorf("You have to specify at least one of the supported flags or specify the services' name")
	}

	if config.DevSpace.Services != nil && len(*config.DevSpace.Services) > 0 {
		newServicesPaths := make([]*v1.ServiceConfig, 0, len(*config.DevSpace.Services)-1)

		for _, v := range *config.DevSpace.Services {
			if removeAll ||
				(name == *v.Name && name != "") ||
				(namespace == *v.Namespace && namespace != "") ||
				(areLabelMapsEqual(labelSelectorMap, *v.LabelSelector) && len(labelSelectorMap) != 0) {
				continue
			}

			newServicesPaths = append(newServicesPaths, v)
		}

		config.DevSpace.Services = &newServicesPaths

		err = configutil.SaveConfig()
		if err != nil {
			return fmt.Errorf("Couldn't save config file: %v", err)
		}
	}

	return nil
}
