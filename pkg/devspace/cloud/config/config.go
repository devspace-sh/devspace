package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
)

// DevSpaceCloudProviderName is the name of the default devspace-cloud provider
const DevSpaceCloudProviderName = "app.devspace.cloud"

// DevSpaceCloudConfigPath holds the path to the cloud config file
const DevSpaceCloudConfigPath = ".devspace/clouds.yaml"

// ReadCloudsConfig reads the cloud config from the home file
func ReadCloudsConfig() ([]byte, error) {
	homedir, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	return ioutil.ReadFile(filepath.Join(homedir, DevSpaceCloudConfigPath))
}

// SaveCloudsConfig saves the cloud config
func SaveCloudsConfig(data []byte) error {
	homedir, err := homedir.Dir()
	if err != nil {
		return err
	}

	cfgPath := filepath.Join(homedir, DevSpaceCloudConfigPath)
	err = os.MkdirAll(filepath.Dir(cfgPath), 0755)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(cfgPath, data, 0600)
}
