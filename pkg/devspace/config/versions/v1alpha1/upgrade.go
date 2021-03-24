package v1alpha1

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/v1alpha2"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"github.com/pkg/errors"
)

// Upgrade upgrades the config
func (c *Config) Upgrade(log log.Logger) (config.Config, error) {
	nextConfig := &next.Config{}
	err := util.Convert(c, nextConfig)
	if err != nil {
		return nil, err
	}

	// Convert deployments
	if c.DevSpace != nil && c.DevSpace.Deployments != nil {
		newConfigDeployments := []*next.DeploymentConfig{}

		for _, deploy := range *c.DevSpace.Deployments {
			newDeployment := &next.DeploymentConfig{
				Name:      deploy.Name,
				Namespace: deploy.Namespace,
			}

			// Add auto reload
			if deploy.AutoReload == nil || deploy.AutoReload.Disabled == nil || *deploy.AutoReload.Disabled {
				if nextConfig.Dev == nil {
					nextConfig.Dev = &next.DevConfig{}
				}
				if nextConfig.Dev.AutoReload == nil {
					nextConfig.Dev.AutoReload = &next.AutoReloadConfig{}
				}
				if nextConfig.Dev.AutoReload.Deployments == nil {
					nextConfig.Dev.AutoReload.Deployments = &[]*string{}
				}

				(*nextConfig.Dev.AutoReload.Deployments) = append((*nextConfig.Dev.AutoReload.Deployments), deploy.Name)
			}

			// Convert kubectl
			if deploy.Kubectl != nil {
				newDeployment.Kubectl = &next.KubectlConfig{
					CmdPath:   deploy.Kubectl.CmdPath,
					Manifests: deploy.Kubectl.Manifests,
				}
			} else if deploy.Helm != nil {
				newDeployment.Helm = &next.HelmConfig{
					ChartPath:      deploy.Helm.ChartPath,
					Wait:           deploy.Helm.Wait,
					OverrideValues: deploy.Helm.OverrideValues,
				}

				if deploy.Helm.DevOverwrite != nil {
					newDeployment.Helm.Overrides = &[]*string{deploy.Helm.DevOverwrite}
				}
				if deploy.Helm.Override != nil {
					newDeployment.Helm.Overrides = &[]*string{deploy.Helm.Override}
				}
			}

			newConfigDeployments = append(newConfigDeployments, newDeployment)
		}

		nextConfig.Deployments = &newConfigDeployments
	}

	// Fill dev config
	if nextConfig.Dev == nil {
		nextConfig.Dev = &next.DevConfig{}
	}

	// Convert devspace to dev
	if c.DevSpace != nil {
		// Convert sync paths
		if c.DevSpace.Sync != nil {
			newSyncPaths := []*next.SyncConfig{}

			for _, sync := range *c.DevSpace.Sync {
				newSyncPaths = append(newSyncPaths, &next.SyncConfig{
					Selector:             sync.Service,
					Namespace:            sync.Namespace,
					LabelSelector:        sync.LabelSelector,
					LocalSubPath:         sync.LocalSubPath,
					ContainerName:        sync.ContainerName,
					ContainerPath:        sync.ContainerPath,
					ExcludePaths:         sync.ExcludePaths,
					DownloadExcludePaths: sync.DownloadExcludePaths,
					UploadExcludePaths:   sync.UploadExcludePaths,
				})

				if sync.BandwidthLimits != nil {
					newSyncPaths[len(newSyncPaths)-1].BandwidthLimits = &next.BandwidthLimits{
						Download: sync.BandwidthLimits.Download,
						Upload:   sync.BandwidthLimits.Upload,
					}
				}
			}

			nextConfig.Dev.Sync = &newSyncPaths
		}

		// Convert ports
		if c.DevSpace.Ports != nil {
			newPorts := []*next.PortForwardingConfig{}

			for _, port := range *c.DevSpace.Ports {
				newPorts = append(newPorts, &next.PortForwardingConfig{
					Selector:      port.Service,
					Namespace:     port.Namespace,
					LabelSelector: port.LabelSelector,
				})

				if port.PortMappings != nil {
					newPortMappings := []*next.PortMapping{}

					for _, portMapping := range *port.PortMappings {
						newPortMappings = append(newPortMappings, &next.PortMapping{
							LocalPort:   portMapping.LocalPort,
							RemotePort:  portMapping.RemotePort,
							BindAddress: portMapping.BindAddress,
						})
					}

					newPorts[len(newPorts)-1].PortMappings = &newPortMappings
				}
			}

			nextConfig.Dev.Ports = &newPorts
		}

		// Convert terminal
		if c.DevSpace.Terminal != nil {
			nextConfig.Dev.Terminal = &next.Terminal{
				Disabled:      c.DevSpace.Terminal.Disabled,
				Selector:      c.DevSpace.Terminal.Service,
				LabelSelector: c.DevSpace.Terminal.LabelSelector,
				Namespace:     c.DevSpace.Terminal.Namespace,
				ContainerName: c.DevSpace.Terminal.ContainerName,
				Command:       c.DevSpace.Terminal.Command,
			}
		}

		// Convert services
		if c.DevSpace.Services != nil {
			selectors := []*next.SelectorConfig{}

			// Convert each service
			for _, service := range *c.DevSpace.Services {
				selectors = append(selectors, &next.SelectorConfig{
					Name:          service.Name,
					Namespace:     service.Namespace,
					LabelSelector: service.LabelSelector,
					ContainerName: service.ContainerName,
				})
			}

			nextConfig.Dev.Selectors = &selectors
		}

		// Convert auto reaload
		if c.DevSpace.AutoReload != nil {
			if nextConfig.Dev.AutoReload == nil {
				nextConfig.Dev.AutoReload = &next.AutoReloadConfig{}
			}

			if c.DevSpace.AutoReload.Paths != nil && len(*c.DevSpace.AutoReload.Paths) > 0 {
				nextConfig.Dev.AutoReload.Paths = c.DevSpace.AutoReload.Paths
			}
		}
	}

	// Convert images with registries
	if c.Images != nil {
		for key, image := range *c.Images {
			(*nextConfig.Images)[key].Image = image.Name

			if image.Registry != nil {
				if c.Registries == nil {
					return nil, errors.Errorf("Registries is nil in config")
				}

				// Get registry
				registry, ok := (*c.Registries)[*image.Registry]
				if ok == false {
					return nil, errors.Errorf("Couldn't find registry %s in registries", *image.Registry)
				}
				if registry.Auth != nil {
					log.Warnf("Registry authentication is not supported any longer (Registry %s). Please use docker login [registry] instead", *image.Registry)
				}
				if registry.URL == nil || image.Name == nil {
					return nil, errors.Errorf("Registry url or image name is nil for image %s", key)
				}

				(*nextConfig.Images)[key].Image = ptr.String(*registry.URL + "/" + *image.Name)
			}

			if image.AutoReload == nil || image.AutoReload.Disabled == nil || *image.AutoReload.Disabled == false {
				if nextConfig.Dev == nil {
					nextConfig.Dev = &next.DevConfig{}
				}
				if nextConfig.Dev.AutoReload == nil {
					nextConfig.Dev.AutoReload = &next.AutoReloadConfig{}
				}
				if nextConfig.Dev.AutoReload.Images == nil {
					nextConfig.Dev.AutoReload.Images = &[]*string{}
				}

				// Assign this because otherwise we get the same value multiple times
				imageName := key
				(*nextConfig.Dev.AutoReload.Images) = append((*nextConfig.Dev.AutoReload.Images), &imageName)
			}
		}
	}

	// Convert tiller namespace
	if c.Tiller != nil && c.Tiller.Namespace != nil {
		if nextConfig.Deployments != nil {
			for _, deploy := range *nextConfig.Deployments {
				if deploy.Helm != nil {
					deploy.Helm.TillerNamespace = c.Tiller.Namespace
				}
			}
		}
	}

	// Convert internal registry
	if c.InternalRegistry != nil {
		log.Warnf("internalRegistry deployment is not supported anymore")
	}

	return nextConfig, nil
}
