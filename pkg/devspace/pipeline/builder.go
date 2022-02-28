package pipeline

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/registry"
	"github.com/loft-sh/devspace/pkg/devspace/devpod"
	"github.com/loft-sh/devspace/pkg/util/tomb"
)

type Builder interface {
	BuildPipeline(name string, devPodManager devpod.Manager, configPipeline *latest.Pipeline, registry registry.DependencyRegistry) (Pipeline, error)
}

func NewPipelineBuilder() Builder {
	return &builder{}
}

type builder struct{}

func (b *builder) BuildPipeline(name string, devPodManager devpod.Manager, configPipeline *latest.Pipeline, registry registry.DependencyRegistry) (Pipeline, error) {
	pip := NewPipeline(name, devPodManager, registry).(*pipeline)
	pip.Job = &Job{
		DevPodManager: pip.DevPodManager(),
		Config:        configPipeline,
		t:             &tomb.Tomb{},
	}

	return pip, nil
}
