package configure

import (
	"fmt"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	v1 "github.com/covexo/devspace/pkg/devspace/config/versions/latest"
	"github.com/covexo/devspace/pkg/devspace/services"
)

// AddSelector adds a selector
func AddSelector(name string, labelSelector string, namespace string, save bool) error {
	config := configutil.GetBaseConfig()

	var labelSelectorMap map[string]*string
	var err error

	if labelSelector == "" {
		config := configutil.GetConfig()

		if config.DevSpace != nil && config.DevSpace.Selectors != nil && len(*config.DevSpace.Selectors) > 0 {
			services := *config.DevSpace.Selectors
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

	if config.DevSpace == nil {
		config.DevSpace = &v1.DevSpaceConfig{}
	}

	if config.DevSpace.Selectors == nil {
		emptyServiceList := make([]*v1.SelectorConfig, 0)
		config.DevSpace.Selectors = &emptyServiceList
	}

	servicesConfig := append(*config.DevSpace.Selectors, &v1.SelectorConfig{
		LabelSelector: &labelSelectorMap,
		Namespace:     &namespace,
		Name:          &name,
	})

	config.DevSpace.Selectors = &servicesConfig

	if save {
		err = configutil.SaveBaseConfig()
		if err != nil {
			return fmt.Errorf("Couldn't save config file: %s", err.Error())
		}
	}

	return nil
}

//RemoveSelector removes a service from the devspace
func RemoveSelector(removeAll bool, name string, labelSelector string, namespace string) error {
	config := configutil.GetBaseConfig()
	labelSelectorMap, err := parseSelectors(labelSelector)

	if err != nil {
		return fmt.Errorf("Error parsing selectors: %v", err)
	}

	if len(labelSelectorMap) == 0 && removeAll == false && name == "" && namespace == "" {
		return fmt.Errorf("You have to specify at least one of the supported flags or specify the selectors' name")
	}

	if config.DevSpace.Selectors != nil && len(*config.DevSpace.Selectors) > 0 {
		newServicesPaths := make([]*v1.SelectorConfig, 0, len(*config.DevSpace.Selectors)-1)

		for _, v := range *config.DevSpace.Selectors {
			if removeAll ||
				(name == *v.Name && name != "") ||
				(namespace == *v.Namespace && namespace != "") ||
				(areLabelMapsEqual(labelSelectorMap, *v.LabelSelector) && len(labelSelectorMap) != 0) {
				continue
			}

			newServicesPaths = append(newServicesPaths, v)
		}

		config.DevSpace.Selectors = &newServicesPaths

		err = configutil.SaveBaseConfig()
		if err != nil {
			return fmt.Errorf("Couldn't save config file: %v", err)
		}
	}

	return nil
}
