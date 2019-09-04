package v1beta2

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/config"
	next "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/util"
	"github.com/devspace-cloud/devspace/pkg/util/log"
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
		log.Warnf("cluster config option is not supported anymore in v1beta2 and devspace v3")
	}

	return nextConfig, nil
}
