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

		if config.Dev != nil && config.Dev.Selectors != nil && len(*config.Dev.Selectors) > 0 {
			services := *config.Dev.Selectors
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

	if config.Dev == nil {
		config.Dev = &v1.DevConfig{}
	}
	if config.Dev.Selectors == nil {
		emptyServiceList := make([]*v1.SelectorConfig, 0)
		config.Dev.Selectors = &emptyServiceList
	}

	servicesConfig := append(*config.Dev.Selectors, &v1.SelectorConfig{
		LabelSelector: &labelSelectorMap,
		Namespace:     &namespace,
		Name:          &name,
	})

	config.Dev.Selectors = &servicesConfig

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

	if config.Dev.Selectors != nil && len(*config.Dev.Selectors) > 0 {
		newServicesPaths := make([]*v1.SelectorConfig, 0, len(*config.Dev.Selectors)-1)

		for _, v := range *config.Dev.Selectors {
			if removeAll ||
				(name == *v.Name && name != "") ||
				(namespace == *v.Namespace && namespace != "") ||
				(areLabelMapsEqual(labelSelectorMap, *v.LabelSelector) && len(labelSelectorMap) != 0) {
				continue
			}

			newServicesPaths = append(newServicesPaths, v)
		}

		config.Dev.Selectors = &newServicesPaths

		err = configutil.SaveBaseConfig()
		if err != nil {
			return fmt.Errorf("Couldn't save config file: %v", err)
		}
	}

	return nil
}
