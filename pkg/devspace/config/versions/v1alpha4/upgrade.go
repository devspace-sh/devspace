package v1alpha4

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/util"
	next "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/v1beta1"
)

// Upgrade upgrades the config
func (c *Config) Upgrade() (config.Config, error) {
	nextConfig := &next.Config{}
	err := util.Convert(c, nextConfig)
	if err != nil {
		return nil, err
	}

	// Convert dockerfilepath and contextpath
	if c.Dev != nil && c.Dev.Ports != nil {
		for key, portConfig := range *c.Dev.Ports {
			if portConfig.PortMappings != nil {
				(*nextConfig.Dev.Ports)[key].PortMappings = &[]*next.PortMapping{}

				for _, portMapping := range *portConfig.PortMappings {
					*(*nextConfig.Dev.Ports)[key].PortMappings = append(*(*nextConfig.Dev.Ports)[key].PortMappings, &next.PortMapping{
						LocalPort:   portMapping.LocalPort,
						RemotePort:  portMapping.RemotePort,
						BindAddress: portMapping.BindAddress,
					})
				}
			}
		}
	}

	return nextConfig, nil
}

// UpgradeVarPaths upgrades the config
func (c *Config) UpgradeVarPaths(varPaths map[string]string) error {
	return nil
}
