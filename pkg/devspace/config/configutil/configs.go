package configutil

import (
	"fmt"

	v1 "github.com/covexo/devspace/pkg/devspace/config/v1"
)

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
		err := loadConfig(returnConfig, *configWrapper.Path)
		if err != nil {
			return nil, fmt.Errorf("Loading config: %v", err)
		}
	} else {
		Merge(&returnConfig, configWrapper.Data)
	}

	return returnConfig, nil
}
