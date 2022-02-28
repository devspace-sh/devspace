package pipeline

import (
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/registry"
	"github.com/loft-sh/devspace/pkg/devspace/devpod"
	"sync"
)

type Pipeline interface {
	Run(ctx *devspacecontext.Context) error
	DevPodManager() devpod.Manager
	Children() []Pipeline
	Name() string
	WaitDev()
}

func NewPipeline(name string, devPodManager devpod.Manager, dependencyRegistry registry.DependencyRegistry) Pipeline {
	return &pipeline{
		name:               name,
		devPodManager:      devPodManager,
		DependencyRegistry: dependencyRegistry,
	}
}

type pipeline struct {
	m sync.Mutex

	name               string
	devPodManager      devpod.Manager
	DependencyRegistry registry.DependencyRegistry

	dependencies []Pipeline
	children     []Pipeline

	Job *Job
}

// WaitDev waits for the dev pod managers to complete.
// This essentially waits until all dev pods are closed.
func (p *pipeline) WaitDev() {
	children := []Pipeline{}
	p.m.Lock()
	children = append(children, p.children...)
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

func (p *pipeline) Children() []Pipeline {
	p.m.Lock()
	defer p.m.Unlock()

	children := []Pipeline{}
	children = append(children, p.children...)
	return children
}

func (p *pipeline) Run(ctx *devspacecontext.Context) error {
	return p.executeJob(ctx, p.Job)
}

func (p *pipeline) executeJob(ctx *devspacecontext.Context, j *Job) error {
	// don't start jobs on a cancelled context
	if ctx.IsDone() {
		return nil
	}

	// set job to completed when done
	err := j.Run(ctx)
	if err != nil {
		return err
	}

	// run children jobs
	return nil
}
