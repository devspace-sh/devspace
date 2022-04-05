package compose

import (
	"path/filepath"

	composetypes "github.com/compose-spec/compose-go/types"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

func (cb *configBuilder) AddDependencies(dockerCompose *composetypes.Project, service composetypes.ServiceConfig) error {
	for _, dependency := range service.GetDependencies() {
		depName := formatName(dependency)

		if cb.config.Dependencies == nil {
			cb.config.Dependencies = map[string]*latest.DependencyConfig{}
		}

		depService, err := dockerCompose.GetService(dependency)
		if err != nil {
			return err
		}

		currentPath := dockerCompose.WorkingDir
		if service.Build != nil && service.Build.Context != "" {
			currentPath = filepath.Join(dockerCompose.WorkingDir, service.Build.Context)
		}

		dependencyPath := dockerCompose.WorkingDir
		if depService.Build != nil && depService.Build.Context != "" {
			dependencyPath = filepath.Join(dockerCompose.WorkingDir, depService.Build.Context)
		}

		relativePath, err := filepath.Rel(currentPath, dependencyPath)
		if err != nil {
			return err
		}

		fileName := ""
		if dependencyPath == dockerCompose.WorkingDir {
			fileName = "devspace-" + depName + ".yaml"
		}

		cb.config.Dependencies[depName] = &latest.DependencyConfig{
			Source: &latest.SourceConfig{
				Path: filepath.ToSlash(filepath.Join(relativePath, fileName)),
			},
		}
	}
	return nil
}
