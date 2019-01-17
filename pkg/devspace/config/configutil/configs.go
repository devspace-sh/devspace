package configutil

import (
	"fmt"
	"io/ioutil"

	v1 "github.com/covexo/devspace/pkg/devspace/config/v1"
	yaml "gopkg.in/yaml.v2"
)

func loadVarsFromWrapper(varsWrapper *v1.VarsWrapper) ([]*v1.Variable, error) {
	if varsWrapper.Path == nil && varsWrapper.Data == nil {
		return nil, fmt.Errorf("path & data key are empty for vars %s", LoadedConfig)
	}
	if varsWrapper.Path != nil && varsWrapper.Data != nil {
		return nil, fmt.Errorf("path & data are both defined in vars %s. Only choose one", LoadedConfig)
	}

	returnVars := []*v1.Variable{}

	// Load from path
	if varsWrapper.Path != nil {
		yamlFileContent, err := ioutil.ReadFile(*varsWrapper.Path)
		if err != nil {
			return nil, fmt.Errorf("Error loading %s: %v", *varsWrapper.Path, err)
		}

		err = yaml.UnmarshalStrict(yamlFileContent, returnVars)
		if err != nil {
			return nil, fmt.Errorf("Error parsing %s: %v", *varsWrapper.Path, err)
		}
	} else {
		returnVars = *varsWrapper.Data
	}

	return returnVars, nil
}

func loadConfigFromWrapper(configWrapper *v1.ConfigWrapper) (*v1.Config, error) {
	if configWrapper.Path == nil && configWrapper.Data == nil {
		return nil, fmt.Errorf("path & data key are empty for config %s", LoadedConfig)
	}
	if configWrapper.Path != nil && configWrapper.Data != nil {
		return nil, fmt.Errorf("path & data are both defined in config %s. Only choose one", LoadedConfig)
	}

	// Config that will be returned
	returnConfig := makeConfig()

	// Load from path
	if configWrapper.Path != nil {
		err := loadConfigFromPath(returnConfig, *configWrapper.Path)
		if err != nil {
			return nil, fmt.Errorf("Loading config: %v", err)
		}
	} else {
		dataConfig := &v1.Config{}

		err := loadConfigFromInterface(dataConfig, configWrapper.Data)
		if err != nil {
			return nil, fmt.Errorf("Loading config from interface: %v", err)
		}

		Merge(&returnConfig, dataConfig)
	}

	return returnConfig, nil
}
