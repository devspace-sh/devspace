package compose

import (
	"path/filepath"

	composetypes "github.com/compose-spec/compose-go/types"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

func (cb *configBuilder) AddImage(dockerCompose *composetypes.Project, service composetypes.ServiceConfig) error {
	build := service.Build
	if build == nil {
		cb.config.Images = nil
		return nil
	}

	contextDir := filepath.Join(dockerCompose.WorkingDir, build.Context)
	context, err := filepath.Rel(cb.workingDir, contextDir)
	if err != nil {
		return err
	}

	context = filepath.ToSlash(context)
	if context == "." {
		context = ""
	}

	dockerfile, err := filepath.Rel(cb.workingDir, filepath.Join(dockerCompose.WorkingDir, build.Context, build.Dockerfile))
	if err != nil {
		return err
	}

	if dockerfile == "Dockerfile" {
		dockerfile = ""
	}

	image := &latest.Image{
		Image:      resolveImage(service),
		Context:    context,
		Dockerfile: filepath.ToSlash(dockerfile),
	}

	if build.Args != nil {
		image.BuildArgs = build.Args
	}

	if build.Target != "" {
		image.Target = build.Target
	}

	if build.Network != "" {
		image.Network = build.Network
	}

	if len(service.Entrypoint) > 0 {
		image.Entrypoint = service.Entrypoint
	}

	if cb.config.Images == nil {
		cb.config.Images = map[string]*latest.Image{}
	}

	imageName := formatName(service.Name)
	cb.config.Images[imageName] = image

	return nil
}

func resolveImage(service composetypes.ServiceConfig) string {
	image := service.Name
	if service.Image != "" {
		image = service.Image
	}
	return image
}
