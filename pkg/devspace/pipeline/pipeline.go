package pipeline

import (
	"context"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/registry"
	"github.com/loft-sh/devspace/pkg/devspace/devpod"
	"sync"
)

func NewPipeline(dependencyRegistry registry.DependencyRegistry, devPodManager devpod.Manager) *Pipeline {
	return &Pipeline{
		DevPodManager:      devPodManager,
		DependencyRegistry: dependencyRegistry,
		JobsPipeline:       []*PipelineJob{},
		openJobs:           make(map[string]*PipelineJob),
		runningJobs:        make(map[string]*PipelineJob),
		completedJobs:      make(map[string]*PipelineJob),
	}
}

type Pipeline struct {
	DevPodManager      devpod.Manager
	DependencyRegistry registry.DependencyRegistry

	Jobs         map[string]*PipelineJob
	JobsPipeline []*PipelineJob

	jobsMutex     sync.Mutex
	openJobs      map[string]*PipelineJob
	runningJobs   map[string]*PipelineJob
	completedJobs map[string]*PipelineJob
}

func (p *Pipeline) Run(ctx *devspacecontext.Context) error {
	for k, v := range p.Jobs {
		p.openJobs[k] = v
	}

	return p.executeJobs(ctx, p.JobsPipeline)
}

func (p *Pipeline) executeJobs(ctx *devspacecontext.Context, jobs []*PipelineJob) error {
	if len(jobs) == 0 {
		return nil
	}

	cancelCtx, cancel := context.WithCancel(ctx.Context)
	defer cancel()
	ctx = ctx.WithContext(cancelCtx)

	waitGroup := sync.WaitGroup{}
	errChan := make(chan error, len(jobs))
	for _, j := range jobs {
		waitGroup.Add(1)

		go func(j *PipelineJob) {
			defer waitGroup.Done()

			err := p.executeJob(ctx, j)
			if err != nil {
				errChan <- err
			}
		}(j)
	}

	done := make(chan struct{})
	go func() {
		waitGroup.Wait()
		close(done)
	}()

	select {
	case err := <-errChan:
		cancel()
		<-done
		return err
	case <-done:
		return nil
	}
}

func (p *Pipeline) executeJob(ctx *devspacecontext.Context, j *PipelineJob) error {
	// don't start jobs on a cancelled context
	select {
	case <-ctx.Context.Done():
		return nil
	default:
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

func (p *Pipeline) setRunning(j *PipelineJob) bool {
	p.jobsMutex.Lock()
	defer p.jobsMutex.Unlock()

	if p.runningJobs[j.Name] != nil || p.completedJobs[j.Name] != nil {
		return true
	}

	delete(p.openJobs, j.Name)
	p.runningJobs[j.Name] = j
	return false
}

func (p *Pipeline) setCompleted(j *PipelineJob) {
	p.jobsMutex.Lock()
	defer p.jobsMutex.Unlock()

	delete(p.runningJobs, j.Name)
	p.completedJobs[j.Name] = j
}
