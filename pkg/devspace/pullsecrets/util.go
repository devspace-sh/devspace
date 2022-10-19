package pullsecrets

import (
	"strings"

	"github.com/docker/distribution/reference"
	dockerregistry "github.com/docker/docker/registry"
)

// GetRegistryFromImageName retrieves the registry name from an imageName
func GetRegistryFromImageName(imageName string) (string, error) {
	ref, err := reference.ParseNormalizedNamed(imageName)
	if err != nil {
		return "", err
	}

	repoInfo, err := dockerregistry.ParseRepositoryInfo(ref)
	if err != nil {
		return "", err
	}

	if repoInfo.Index.Official {
		return "", nil
	}

	return repoInfo.Index.Name, nil
}

func IsAzureContainerRegistry(serverAddress string) bool {
	return strings.HasSuffix(serverAddress, "azurecr.io")
}
