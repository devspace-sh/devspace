package configutil

import (
	"fmt"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configs"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
)

func loadVarsFromWrapper(varsWrapper *configs.VarsWrapper) ([]*configs.Variable, error) {
	if varsWrapper.Path == nil && varsWrapper.Data == nil {
		return nil, fmt.Errorf("path & data key are empty for vars %s", LoadedConfig)
	}
	if varsWrapper.Path != nil && varsWrapper.Data != nil {
		return nil, fmt.Errorf("path & data are both defined in vars %s. Only choose one", LoadedConfig)
	}

	returnVars := []*configs.Variable{}

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

func loadConfigFromWrapper(configWrapper *configs.ConfigWrapper) (*latest.Config, error) {
	if configWrapper.Path == nil && configWrapper.Data == nil {
		return nil, fmt.Errorf("path & data key are empty for config %s", LoadedConfig)
	}
	if configWrapper.Path != nil && configWrapper.Data != nil {
		return nil, fmt.Errorf("path & data are both defined in config %s. Only choose one", LoadedConfig)
	}

	// Config that will be returned
	var err error
	var returnConfig *latest.Config

	// Load from path
	if configWrapper.Path != nil {
		returnConfig, err = loadConfigFromPath(*configWrapper.Path)
		if err != nil {
			return nil, fmt.Errorf("Loading config: %v", err)
		}
	} else {
		returnConfig, err = loadConfigFromInterface(configWrapper.Data)
		if err != nil {
			return nil, fmt.Errorf("Loading config from interface: %v", err)
		}
	}

	return returnConfig, nil
}
