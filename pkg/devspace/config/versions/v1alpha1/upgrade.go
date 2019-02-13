package v1alpha1

import (
	"fmt"

	"github.com/covexo/devspace/pkg/devspace/config/versions/config"
	next "github.com/covexo/devspace/pkg/devspace/config/versions/latest"
	"github.com/covexo/devspace/pkg/devspace/config/versions/util"
	"github.com/covexo/devspace/pkg/util/ptr"
)

// Upgrade upgrades the config
func (c *Config) Upgrade() (config.Config, error) {
	nextConfig := &next.Config{}
	err := util.Convert(c, nextConfig)
	if err != nil {
		return nil, err
	}

	// Convert helm devOverwrites and override
	if c.DevSpace != nil && c.DevSpace.Deployments != nil {
		for idx, deploy := range *c.DevSpace.Deployments {
			if deploy.Helm != nil {
				if deploy.Helm.DevOverwrite != nil {
					(*nextConfig.DevSpace.Deployments)[idx].Helm.Overrides = &[]*string{deploy.Helm.DevOverwrite}
				}
				if deploy.Helm.Override != nil {
					(*nextConfig.DevSpace.Deployments)[idx].Helm.Overrides = &[]*string{deploy.Helm.Override}
				}
			}
		}
	}

	// Convert services
	if c.DevSpace != nil {
		// Convert sync paths
		if c.DevSpace.Sync != nil {
			for idx, sync := range *c.DevSpace.Sync {
				if sync.Service != nil {
					(*nextConfig.DevSpace.Sync)[idx].Selector = sync.Service
				}
			}
		}

		// Convert ports
		if c.DevSpace.Ports != nil {
			for idx, port := range *c.DevSpace.Ports {
				if port.Service != nil {
					(*nextConfig.DevSpace.Ports)[idx].Selector = port.Service
				}
			}
		}

		// Convert terminal
		if c.DevSpace.Terminal != nil {
			if c.DevSpace.Terminal.Service != nil {
				nextConfig.DevSpace.Terminal.Selector = c.DevSpace.Terminal.Service
			}
		}

		// Convert services
		if c.DevSpace.Services != nil {
			if nextConfig.DevSpace == nil {
				nextConfig.DevSpace = &next.DevSpaceConfig{}
			}

			selectors := []*next.SelectorConfig{}

			// Convert each service
			for _, service := range *c.DevSpace.Services {
				selectors = append(selectors, &next.SelectorConfig{
					Name:          service.Name,
					Namespace:     service.Namespace,
					ResourceType:  service.ResourceType,
					LabelSelector: service.LabelSelector,
					ContainerName: service.ContainerName,
				})
			}

			nextConfig.DevSpace.Selectors = &selectors
		}
	}

	// Convert registries
	if c.Images != nil {
		for key, image := range *c.Images {
			if image.Registry != nil {
				if c.Registries == nil {
					return nil, fmt.Errorf("Registries is nil in config")
				}

				// Get registry
				registry, ok := (*c.Registries)[*image.Registry]
				if ok == false {
					return nil, fmt.Errorf("Couldn't find registry %s in registries", *image.Registry)
				}
				if registry.Auth != nil {
					return nil, fmt.Errorf("Registry authentication is not supported any longer (Registry %s). Please use docker login [registry] instead", *image.Registry)
				}
				if registry.URL == nil || image.Name == nil {
					return nil, fmt.Errorf("Registry url or image name is nil for image %s", key)
				}

				(*nextConfig.Images)[key].Name = ptr.String(*registry.URL + "/" + *image.Name)
			}
		}
	}

	// Convert tiller namespace
	if c.Tiller != nil && c.Tiller.Namespace != nil {
		if nextConfig.DevSpace != nil && nextConfig.DevSpace.Deployments != nil {
			for _, deploy := range *nextConfig.DevSpace.Deployments {
				if deploy.Helm != nil {
					deploy.Helm.TillerNamespace = c.Tiller.Namespace
				}
			}
		}
	}

	// Convert internal registry
	if c.InternalRegistry != nil {
		return nil, fmt.Errorf("internalRegistry deployment is not supported anymore")
	}

	return nextConfig, nil
}
