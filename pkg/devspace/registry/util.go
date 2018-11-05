package registry

import (
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/docker/distribution/reference"
	dockerregistry "github.com/docker/docker/registry"
)

// GetRegistryConfigFromImageConfig from image config returns a registry config from an image config
func GetRegistryConfigFromImageConfig(imageConf *v1.ImageConfig) (*v1.RegistryConfig, error) {
	imageName := *imageConf.Name
	registryConf := &v1.RegistryConfig{
		URL:      configutil.String(""),
		Insecure: configutil.Bool(false),
	}

	if imageConf.Registry != nil {
		oldRegistryConf, err := GetRegistryConfig(imageConf)
		if err != nil {
			return nil, err
		}

		if oldRegistryConf.URL != nil {
			*registryConf.URL = *oldRegistryConf.URL
		}
		if *registryConf.URL == "hub.docker.com" {
			*registryConf.URL = ""
		}
	} else {
		registryURL, err := GetRegistryFromImageName(imageName)
		if err != nil {
			return nil, err
		}

		if len(registryURL) > 0 {
			// Crop registry Url from imageName
			imageName = imageName[len(registryURL)+1:]
		}

		registryConf.URL = &registryURL
	}

	if *registryConf.URL != "" {
		// Check if it's the official registry or not
		ref, err := reference.ParseNormalizedNamed(*registryConf.URL + "/" + imageName)
		if err != nil {
			return nil, err
		}

		repoInfo, err := dockerregistry.ParseRepositoryInfo(ref)
		if err != nil {
			return nil, err
		}

		if repoInfo.Index.Official == true {
			registryConf.URL = configutil.String("")
		}
	}

	return registryConf, nil
}

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
