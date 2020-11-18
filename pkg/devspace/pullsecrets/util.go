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

// GetStrippedDockerImageName returns a tag stripped image name and checks if it's a valid image name
func GetStrippedDockerImageName(imageName string) (string, error) {
	imageName = strings.TrimSpace(imageName)

	// Check if we can parse the name
	ref, err := reference.ParseNormalizedNamed(imageName)
	if err != nil {
		return "", err
	}

	repoInfo, err := dockerregistry.ParseRepositoryInfo(ref)
	if err != nil {
		return "", err
	}

	if repoInfo.Index.Official {
		// strip docker.io and library from image
		return strings.TrimPrefix(strings.TrimPrefix(reference.TrimNamed(ref).Name(), repoInfo.Index.Name+"/library/"), repoInfo.Index.Name+"/"), nil
	}

	return reference.TrimNamed(ref).Name(), nil
}
