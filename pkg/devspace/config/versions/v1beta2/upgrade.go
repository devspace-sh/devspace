package v1beta2

import (
	"errors"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta3"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/ptr"
)

// getSelector returns the service referenced by serviceName
func getSelector(config *Config, selectorName string) (*SelectorConfig, error) {
	if config.Dev.Selectors != nil {
		for _, selector := range *config.Dev.Selectors {
			if *selector.Name == selectorName {
				return selector, nil
			}
		}
	}

	return nil, errors.New("Unable to find selector: " + selectorName)
}

func ptrArrToStrArr(ptrArr *[]*string) []string {
	if ptrArr == nil {
		return nil
	}

	retArr := []string{}
	for _, v := range *ptrArr {
		retArr = append(retArr, *v)
	}

	return retArr
}

func ptrMapToStrMap(ptrMap *map[string]*string) map[string]string {
	if ptrMap == nil {
		return nil
	}

	retMap := make(map[string]string)
	for k, v := range *ptrMap {
		retMap[k] = *v
	}

	return retMap
}

// Upgrade upgrades the config
func (c *Config) Upgrade(log log.Logger) (config.Config, error) {
	nextConfig := &next.Config{}
	err := util.Convert(c, nextConfig)
	if err != nil {
		return nil, err
	}

	// Check if old cluster exists
	if c.Cluster != nil && (c.Cluster.KubeContext != nil || c.Cluster.Namespace != nil) {
		log.Warnf("cluster config option is not supported anymore in v1beta2 and devspace v4")
	}

	if nextConfig.Dev == nil {
		nextConfig.Dev = &next.DevConfig{}
	}
	if nextConfig.Dev.Interactive == nil {
		nextConfig.Dev.Interactive = &next.InteractiveConfig{}
	}

	if c.Dev != nil && c.Dev.Terminal != nil && c.Dev.Terminal.Disabled != nil {
		nextConfig.Dev.Interactive.DefaultEnabled = ptr.Bool(!*c.Dev.Terminal.Disabled)
	} else {
		nextConfig.Dev.Interactive.DefaultEnabled = ptr.Bool(true)
	}

	// Convert override images
	if c.Dev != nil {
		if c.Dev.OverrideImages != nil && len(*c.Dev.OverrideImages) > 0 {
			nextConfig.Dev.Interactive.Images = []*next.InteractiveImageConfig{}
			for _, overrideImage := range *c.Dev.OverrideImages {
				if overrideImage.Name == nil {
					continue
				}
				if overrideImage.Dockerfile != nil {
					log.Warnf("dev.overrideImages[*].dockerfile is not supported anymore, please use profiles instead")
				}
				if overrideImage.Context != nil {
					log.Warnf("dev.overrideImages[*].context is not supported anymore, please use profiles instead")
				}

				entrypoint := []string{}
				cmd := []string{}
				if overrideImage.Entrypoint != nil && len(*overrideImage.Entrypoint) > 0 {
					entrypoint = []string{*(*overrideImage.Entrypoint)[0]}

					for i, s := range *overrideImage.Entrypoint {
						if i == 0 {
							continue
						}

						cmd = append(cmd, *s)
					}
				}

				nextConfig.Dev.Interactive.Images = append(nextConfig.Dev.Interactive.Images, &next.InteractiveImageConfig{
					Name:       *overrideImage.Name,
					Entrypoint: entrypoint,
					Cmd:        cmd,
				})
			}
		}

		// Convert terminal
		if c.Dev.Terminal != nil {
			if c.Dev.Terminal.Command != nil || c.Dev.Terminal.Selector != nil || c.Dev.Terminal.LabelSelector != nil || c.Dev.Terminal.Namespace != nil || c.Dev.Terminal.ContainerName != nil {
				if c.Dev.Terminal.Selector != nil {
					selector, err := getSelector(c, *c.Dev.Terminal.Selector)
					if err != nil {
						return nil, err
					}

					nextConfig.Dev.Interactive.Terminal = &next.TerminalConfig{
						LabelSelector: ptrMapToStrMap(selector.LabelSelector),
						Namespace:     ptr.ReverseString(selector.Namespace),
						ContainerName: ptr.ReverseString(selector.ContainerName),
					}
				} else {
					nextConfig.Dev.Interactive.Terminal = &next.TerminalConfig{
						LabelSelector: ptrMapToStrMap(c.Dev.Terminal.LabelSelector),
						Namespace:     ptr.ReverseString(c.Dev.Terminal.Namespace),
						ContainerName: ptr.ReverseString(c.Dev.Terminal.ContainerName),
					}
				}

				nextConfig.Dev.Interactive.Terminal.Command = ptrArrToStrArr(c.Dev.Terminal.Command)
			}
		}

		// Convert sync
		if c.Dev.Sync != nil {
			for idx, syncConfig := range *c.Dev.Sync {
				if syncConfig.Selector != nil {
					selector, err := getSelector(c, *syncConfig.Selector)
					if err != nil {
						return nil, err
					}

					if selector.LabelSelector != nil {
						nextConfig.Dev.Sync[idx].LabelSelector = ptrMapToStrMap(selector.LabelSelector)
					}
					if selector.Namespace != nil {
						nextConfig.Dev.Sync[idx].Namespace = *selector.Namespace
					}
					if selector.ContainerName != nil {
						nextConfig.Dev.Sync[idx].ContainerName = *selector.ContainerName
					}
				}
			}
		}

		// Convert port forward
		if c.Dev.Ports != nil {
			for idx, portConfig := range *c.Dev.Ports {
				if portConfig.Selector != nil {
					selector, err := getSelector(c, *portConfig.Selector)
					if err != nil {
						return nil, err
					}

					if selector.LabelSelector != nil {
						nextConfig.Dev.Ports[idx].LabelSelector = ptrMapToStrMap(selector.LabelSelector)
					}
					if selector.Namespace != nil {
						nextConfig.Dev.Ports[idx].Namespace = *selector.Namespace
					}
				}
			}
		}
	}

	// Upgrade dependencies
	if c.Dependencies != nil {
		for idx, dependency := range *c.Dependencies {
			if dependency.Config != nil {
				nextConfig.Dependencies[idx].Profile = *dependency.Config
			}
		}
	}

	// Upgrade images
	if c.Images != nil {
		for imageName, image := range *c.Images {
			if image.CreatePullSecret == nil {
				nextConfig.Images[imageName].CreatePullSecret = ptr.Bool(false)
			}
		}
	}

	// Update deployments
	if c.Deployments != nil {
		for idx, deployment := range *c.Deployments {
			if deployment.Helm != nil {
				nextConfig.Deployments[idx].Helm.ReplaceImageTags = deployment.Helm.DevSpaceValues
			}
		}
	}

	return nextConfig, nil
}
