package pipeline

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/registry"
	"github.com/loft-sh/devspace/pkg/devspace/devpod"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"sync"
)

func NewPipeline(name string, devPodManager devpod.Manager, dependencyRegistry registry.DependencyRegistry) types.Pipeline {
	return &pipeline{
		name:               name,
		devPodManager:      devPodManager,
		dependencyRegistry: dependencyRegistry,
		jobs:               make(map[string]*Job),
	}
}

type pipeline struct {
	m sync.Mutex

	name               string
	devPodManager      devpod.Manager
	dependencyRegistry registry.DependencyRegistry

	dependencies []types.Pipeline

	main *Job
	jobs map[string]*Job
}

// WaitDev waits for the dev pod managers to complete.
// This essentially waits until all dev pods are closed.
func (p *pipeline) WaitDev() {
	children := []types.Pipeline{}
	p.m.Lock()
	children = append(children, p.dependencies...)
	p.m.Unlock()

	// wait for children first
	for _, child := range children {
		child.WaitDev()
	}

	// wait for dev pods to finish
	p.devPodManager.Wait()
}

func (p *pipeline) Name() string {
	return p.name
}

func (p *pipeline) DevPodManager() devpod.Manager {
	return p.devPodManager
}

func (p *pipeline) DependencyRegistry() registry.DependencyRegistry {
	return p.dependencyRegistry
}

func (p *pipeline) Dependencies() []types.Pipeline {
	p.m.Lock()
	defer p.m.Unlock()

	children := []types.Pipeline{}
	children = append(children, p.dependencies...)
	return children
}

func (p *pipeline) Run(ctx *devspacecontext.Context) error {
	return p.executeJob(ctx, p.main)
}

func (p *pipeline) StartNewPipelines(ctx *devspacecontext.Context, pipelines []*latest.Pipeline, sequentially bool) error {
	if sequentially {
		for _, configPipeline := range pipelines {
			err := p.StartNewPipeline(ctx, configPipeline)
			if err != nil {
				return err
			}
		}

		return nil
	}

	// Start concurrently
	ctx, t := ctx.WithNewTomb()
	t.Go(func() error {
		for _, configPipeline := range pipelines {
			func(configPipeline *latest.Pipeline) {
				t.Go(func() error {
					return p.StartNewPipeline(ctx, configPipeline)
				})
			}(configPipeline)
		}
		return nil
	})

	return t.Wait()
}

func (p *pipeline) StartNewPipeline(ctx *devspacecontext.Context, configPipeline *latest.Pipeline) error {
	if configPipeline.Name == p.name {
		return fmt.Errorf("pipeline %s is already running", configPipeline.Name)
	}

	// exchange job if it's not alive anymore
	p.m.Lock()
	j, ok := p.jobs[configPipeline.Name]
	if ok && !j.Terminated() {
		p.m.Unlock()
		return fmt.Errorf("pipeline %s is already running", configPipeline.Name)
	}

	j = &Job{
		Pipeline: p,
		Config:   configPipeline,
	}
	p.jobs[configPipeline.Name] = j
	p.m.Unlock()

	return p.executeJob(ctx, j)
}

func (p *pipeline) executeJob(ctx *devspacecontext.Context, j *Job) error {
	// don't start jobs on a cancelled context
	if ctx.IsDone() {
		return nil
	}

	err := j.Run(ctx)
	if err != nil {
		return err
	}

	return nil
}
