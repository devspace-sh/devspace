package v1beta3

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta4"
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

	if len(c.Deployments) > 0 {
		for idx, deploymentConfig := range c.Deployments {
			if deploymentConfig.Component != nil {
				helmValues := map[string]interface{}{}
				err = util.Convert(deploymentConfig.Component, &helmValues)
				if err != nil {
					return nil, err
				}

				delete(helmValues, "options")
				if deploymentConfig.Component.Options == nil {
					deploymentConfig.Component.Options = &ComponentConfigOptions{}
				}

				nextConfig.Deployments[idx].Helm = &next.HelmConfig{
					ComponentChart:   ptr.Bool(true),
					Values:           helmValues,
					ReplaceImageTags: deploymentConfig.Component.Options.ReplaceImageTags,
					Force:            deploymentConfig.Component.Options.Force,
					Wait:             deploymentConfig.Component.Options.Wait,
					Timeout:          deploymentConfig.Component.Options.Timeout,
					Rollback:         deploymentConfig.Component.Options.Rollback,
					TillerNamespace:  deploymentConfig.Component.Options.TillerNamespace,
				}
			}

			if deploymentConfig.Helm != nil && deploymentConfig.Helm.Chart != nil && deploymentConfig.Helm.Chart.Name == "component-chart" && deploymentConfig.Helm.Chart.RepoURL == "https://charts.devspace.cloud" && deploymentConfig.Helm.Chart.Version == "v0.0.6" {
				nextConfig.Deployments[idx].Helm.Chart = nil
				nextConfig.Deployments[idx].Helm.ComponentChart = ptr.Bool(true)
			}
		}
	}

	return nextConfig, nil
}
