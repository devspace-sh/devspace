package docker

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	dockerregistry "github.com/docker/docker/api/types/registry"

	"github.com/docker/docker/pkg/homedir"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
)

const dockerFileFolder = ".docker"

var configDir = os.Getenv("DOCKER_CONFIG")

var configDirOnce sync.Once

func LoadDockerConfig() (*configfile.ConfigFile, error) {
	configDirOnce.Do(func() {
		if configDir == "" {
			configDir = filepath.Join(homedir.Get(), dockerFileFolder)
		}
	})

	return config.Load(configDir)
}

// GetAllAuthConfigs returns every auth config found in the docker config
func GetAllAuthConfigs() (map[string]dockerregistry.AuthConfig, error) {
	config, err := LoadDockerConfig()
	if err != nil {
		return nil, err
	}

	authMap, err := config.GetAllCredentials()
	if err != nil {
		return nil, err
	}

	retMap := make(map[string]dockerregistry.AuthConfig)
	for k, v := range authMap {
		// convert
		authconfigConverted := &dockerregistry.AuthConfig{}
		err = util.Convert(v, authconfigConverted)
		if err != nil {
			return nil, err
		}

		retMap[k] = *authconfigConverted
	}

	return retMap, nil
}
