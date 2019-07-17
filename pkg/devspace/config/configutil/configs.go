package configutil

import (
	"fmt"
	"path/filepath"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configs"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
)

func loadConfigFromWrapper(basePath string, configWrapper *configs.ConfigWrapper) (*latest.Config, error) {
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
		returnConfig, err = loadConfigFromPath(filepath.Join(basePath, filepath.FromSlash(*configWrapper.Path)))
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
