package compose

import (
	composetypes "github.com/compose-spec/compose-go/types"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

func (cb *configBuilder) AddDependencies(dockerCompose composetypes.Project, service composetypes.ServiceConfig) error {
	for _, dependency := range service.GetDependencies() {
		if cb.config.Dependencies == nil {
			cb.config.Dependencies = map[string]*latest.DependencyConfig{}
		}

		depName := formatName(dependency)
		cb.config.Dependencies[depName] = &latest.DependencyConfig{
			Source: &latest.SourceConfig{
				Path: "devspace-" + depName + ".yaml",
			},
		}
	}
	return nil
}
