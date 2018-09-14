package docker

import (
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/homedir"
)

const dockerFileFolder = ".docker"

var configDir = os.Getenv("DOCKER_CONFIG")

func init() {
	if configDir == "" {
		configDir = filepath.Join(homedir.Get(), dockerFileFolder)
	}
}

func loadDockerConfig() (*configfile.ConfigFile, error) {
	return config.Load(configDir)
}

func getAllAuthConfigs() (map[string]types.AuthConfig, error) {
	config, err := loadDockerConfig()
	if err != nil {
		return nil, err
	}

	return config.GetAllCredentials()
}
