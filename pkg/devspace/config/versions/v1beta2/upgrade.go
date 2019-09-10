package v1beta2

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/config"
	next "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/util"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
)

// Upgrade upgrades the config
func (c *Config) Upgrade() (config.Config, error) {
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
		nextConfig.Dev.Interactive.Enabled = ptr.Bool(!*c.Dev.Terminal.Disabled)
	} else {
		nextConfig.Dev.Interactive.Enabled = ptr.Bool(true)
	}

	// Convert override images
	if c.Dev != nil && c.Dev.OverrideImages != nil && len(*c.Dev.OverrideImages) > 0 {
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
