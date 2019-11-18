package docker

import (
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/util"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/homedir"
)

const dockerFileFolder = ".docker"

var configDir = os.Getenv("DOCKER_CONFIG")

func loadDockerConfig() (*configfile.ConfigFile, error) {
	if configDir == "" {
		configDir = filepath.Join(homedir.Get(), dockerFileFolder)
	}

	return config.Load(configDir)
}

// GetAllAuthConfigs returns every auth config found in the docker config
func GetAllAuthConfigs() (map[string]types.AuthConfig, error) {
	config, err := loadDockerConfig()
	if err != nil {
		return nil, err
	}

	authMap, err := config.GetAllCredentials()
	if err != nil {
		return nil, err
	}

	retMap := make(map[string]types.AuthConfig)
	for k, v := range authMap {
		// convert
		authconfigConverted := &types.AuthConfig{}
		err = util.Convert(v, authconfigConverted)
		if err != nil {
			return nil, err
		}

		retMap[k] = *authconfigConverted
	}

	return retMap, nil
}
