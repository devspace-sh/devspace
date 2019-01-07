package configutil

import (
	v1 "github.com/covexo/devspace/pkg/devspace/config/v1"
)

func makeConfig() *v1.Config {
	return &v1.Config{
		Cluster: &v1.Cluster{
			User: &v1.ClusterUser{},
		},
		DevSpace: &v1.DevSpaceConfig{
			Terminal: &v1.Terminal{},
		},
		Images:     &map[string]*v1.ImageConfig{},
		Registries: &map[string]*v1.RegistryConfig{},
	}
}
