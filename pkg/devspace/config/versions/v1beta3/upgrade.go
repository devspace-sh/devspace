package v1beta3

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/config"
	next "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/util"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
)

// Upgrade upgrades the config
func (c *Config) Upgrade() (config.Config, error) {
	nextConfig := &next.Config{}
	err := util.Convert(c, nextConfig)
	if err != nil {
		return nil, err
	}

	if len(c.Deployments) > 0 {
		for idx, deploymentConfig := range c.Deployments {
			if deploymentConfig.Component != nil {
				helmValues := map[interface{}]interface{}{}
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
