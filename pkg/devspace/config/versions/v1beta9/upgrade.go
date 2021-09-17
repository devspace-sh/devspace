package v1beta9

import (
	"strconv"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta10"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/ptr"
)

// Upgrade upgrades the config
func (c *Config) Upgrade(log log.Logger) (config.Config, error) {
	nextConfig := &next.Config{}
	err := util.Convert(c, nextConfig)
	if err != nil {
		return nil, err
	}

	// convert dependencies
	for k, d := range c.Dependencies {
		if d.Name == "" {
			nextConfig.Dependencies[k].Name = strconv.Itoa(k)
		}
	}

	// convert images
	for k, m := range c.Images {
		if m.TagsAppendRandom {
			for i := range nextConfig.Images[k].Tags {
				nextConfig.Images[k].Tags[i] = nextConfig.Images[k].Tags[i] + "-#####"
			}
		}

		// check if there is a sync config for this image
		if m.PreferSyncOverRebuild {
			found := false
			if c.Dev != nil {
				for _, s := range c.Dev.Sync {
					if s.ImageName == k {
						found = true
						break
					}
				}
			}
			if found {
				nextConfig.Images[k].RebuildStrategy = next.RebuildStrategyIgnoreContextChanges
			}
		}
	}

	// convert dev
	if c.Dev != nil {
		for i, s := range c.Dev.Sync {
			// wait initial sync changed to default enabled
			if s.WaitInitialSync == nil {
				nextConfig.Dev.Sync[i].WaitInitialSync = ptr.Bool(false)
			}
			nextConfig.Dev.Sync[i].Polling = true
		}

		if c.Dev.Interactive != nil {
			// set terminal options
			if c.Dev.Interactive.Terminal != nil {
				nextConfig.Dev.Terminal = &next.Terminal{}
				nextConfig.Dev.Terminal.ImageName = c.Dev.Interactive.Terminal.ImageName
				nextConfig.Dev.Terminal.LabelSelector = c.Dev.Interactive.Terminal.LabelSelector
				nextConfig.Dev.Terminal.ContainerName = c.Dev.Interactive.Terminal.ContainerName
				nextConfig.Dev.Terminal.Namespace = c.Dev.Interactive.Terminal.Namespace
				nextConfig.Dev.Terminal.Command = c.Dev.Interactive.Terminal.Command
				nextConfig.Dev.Terminal.WorkDir = c.Dev.Interactive.Terminal.WorkDir
				if c.Dev.Interactive.DefaultEnabled == nil || !*c.Dev.Interactive.DefaultEnabled {
					nextConfig.Dev.Terminal.Disabled = true
				}
			}

			// is disabled by default?
			if c.Dev.Interactive.DefaultEnabled != nil && *c.Dev.Interactive.DefaultEnabled {
				nextConfig.Dev.InteractiveEnabled = true
			}

			// set deprecated interactive images
			if c.Dev.Interactive.Images != nil {
				nextConfig.Dev.InteractiveImages = []*next.InteractiveImageConfig{}
				for _, img := range c.Dev.Interactive.Images {
					nextConfig.Dev.InteractiveImages = append(nextConfig.Dev.InteractiveImages, &next.InteractiveImageConfig{
						Name:       img.Name,
						Entrypoint: img.Entrypoint,
						Cmd:        img.Cmd,
					})
				}
			}

		}
	}

	return nextConfig, nil
}
