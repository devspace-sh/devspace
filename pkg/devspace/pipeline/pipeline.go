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
	WaitDev()
}

func NewPipeline(devPodManager devpod.Manager, dependencyRegistry registry.DependencyRegistry) Pipeline {
	return &pipeline{
		devPodManager:      devPodManager,
		DependencyRegistry: dependencyRegistry,
		JobsPipeline:       []*PipelineJob{},
		openJobs:           make(map[string]*PipelineJob),
		runningJobs:        make(map[string]*PipelineJob),
		completedJobs:      make(map[string]*PipelineJob),
	}
}

type pipeline struct {
	m sync.Mutex

	devPodManager      devpod.Manager
	DependencyRegistry registry.DependencyRegistry

	children []Pipeline

	Jobs         map[string]*PipelineJob
	JobsPipeline []*PipelineJob

	openJobs      map[string]*PipelineJob
	runningJobs   map[string]*PipelineJob
	completedJobs map[string]*PipelineJob
}

// WaitDev waits for the dev pod managers to complete.
// This essentially waits until all dev pods are closed.
func (p *pipeline) WaitDev() {
	children := []Pipeline{}
	p.m.Lock()
	children = append(children, p.children...)
	p.m.Unlock()

	// wait for children first
	for _, child := range children {
		child.WaitDev()
	}

	// wait for dev pods to finish
	p.devPodManager.Wait()
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
	for k, v := range p.Jobs {
		p.openJobs[k] = v
	}

	return p.executeJobs(ctx, p.JobsPipeline)
}

func (p *pipeline) executeJobs(ctx *devspacecontext.Context, jobs []*PipelineJob) error {
	if len(jobs) == 0 {
		return nil
	}

	ctx, t := ctx.WithNewTomb()
	t.Go(func() error {
		for _, j := range jobs {
			func(j *PipelineJob) {
				t.Go(func() error {
					return p.executeJob(ctx, j)
				})
			}(j)
		}

		return nil
	})

	return t.Wait()
}

func (p *pipeline) executeJob(ctx *devspacecontext.Context, j *PipelineJob) error {
	// don't start jobs on a cancelled context
	if ctx.IsDone() {
		return nil
	}

	// make sure nobody else if running this job already
	alreadyRunning := p.setRunning(j)
	if alreadyRunning {
		return nil
	}

	// set job to completed when done
	err := j.Run(ctx)
	p.setCompleted(j)
	if err != nil {
		return err
	}

	// run children jobs
	return p.executeJobs(ctx, j.Children)
}

func (p *pipeline) setRunning(j *PipelineJob) bool {
	p.m.Lock()
	defer p.m.Unlock()

	if p.runningJobs[j.Name] != nil || p.completedJobs[j.Name] != nil {
		return true
	}

	delete(p.openJobs, j.Name)
	p.runningJobs[j.Name] = j
	return false
}

func (p *pipeline) setCompleted(j *PipelineJob) {
	p.m.Lock()
	defer p.m.Unlock()

	delete(p.runningJobs, j.Name)
	p.completedJobs[j.Name] = j
}
