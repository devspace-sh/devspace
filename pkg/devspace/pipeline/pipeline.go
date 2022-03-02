package pipeline

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/registry"
	types2 "github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/devpod"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/stringutil"
	"github.com/pkg/errors"
	"sync"
)

func NewPipeline(name string, devPodManager devpod.Manager, dependencyRegistry registry.DependencyRegistry, config *latest.Pipeline, options types.Options) types.Pipeline {
	pip := &pipeline{
		name:               name,
		devPodManager:      devPodManager,
		dependencyRegistry: dependencyRegistry,
		options:            options,
		jobs:               make(map[string]*Job),
	}
	pip.main = &Job{
		Pipeline: pip,
		Config:   config,
	}
	return pip
}

type pipeline struct {
	m sync.Mutex

	options types.Options

	name               string
	devPodManager      devpod.Manager
	dependencyRegistry registry.DependencyRegistry

	dependencies []types.Pipeline

	main *Job
	jobs map[string]*Job
}

func (p *pipeline) Options() types.Options {
	return p.options
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

func (p *pipeline) StartNewDependencies(ctx *devspacecontext.Context, dependencies []types2.Dependency, options types.DependencyOptions) error {
	dependencyNames := []string{}
	for _, dependency := range dependencies {
		dependencyNames = append(dependencyNames, dependency.Name())
	}

	deployableDependencies, err := p.dependencyRegistry.MarkDependenciesExcluded(ctx, dependencyNames, false)
	if err != nil {
		return errors.Wrap(err, "check if dependencies can be deployed")
	}

	deployDependencies := []types2.Dependency{}
	for _, dependency := range dependencies {
		if stringutil.Contains(options.Exclude, dependency.Name()) {
			ctx.Log.Debugf("Skipping dependency %s because it was excluded", dependency.Name())
			continue
		} else if !deployableDependencies[dependency.Name()] {
			ctx.Log.Infof("Skipping dependency %s as it was either already deployed or is currently in use by another DevSpace instance in the same namespace", dependency.Name())
			continue
		}

		deployDependencies = append(deployDependencies, dependency)
	}

	if options.Sequential {
		for _, dependency := range deployDependencies {
			err := p.startNewDependency(ctx, dependency, options)
			if err != nil {
				return errors.Wrapf(err, "run dependency %s", dependency.Name())
			}
		}

		return nil
	}

	// Start concurrently
	ctx, t := ctx.WithNewTomb()
	t.Go(func() error {
		for _, dependency := range deployDependencies {
			func(dependency types2.Dependency) {
				t.Go(func() error {
					return p.startNewDependency(ctx, dependency, options)
				})
			}(dependency)
		}
		return nil
	})

	return t.Wait()
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

func (p *pipeline) startNewDependency(ctx *devspacecontext.Context, dependency types2.Dependency, options types.DependencyOptions) error {
	// find the dependency pipeline to execute
	pipeline := options.Pipeline
	if pipeline == "" {
		if dependency.DependencyConfig().Pipeline != "" {
			pipeline = dependency.DependencyConfig().Pipeline
		} else {
			pipeline = p.name
		}
	}

	// find pipeline
	var (
		pipelineConfig *latest.Pipeline
		err            error
	)
	if dependency.Config().Config().Pipelines == nil || dependency.Config().Config().Pipelines[pipeline] == nil {
		pipelineConfig, err = GetDefaultPipeline(pipeline)
		if err != nil {
			return err
		}
	} else {
		pipelineConfig = dependency.Config().Config().Pipelines[pipeline]
	}

	dependencyDevPodManager := devpod.NewManager(p.devPodManager.Context())
	pip := NewPipeline(dependency.Name(), dependencyDevPodManager, p.dependencyRegistry, pipelineConfig, p.options)

	p.m.Lock()
	p.dependencies = append(p.dependencies, pip)
	p.m.Unlock()

	ctx = ctx.WithLogger(log.NewDefaultPrefixLogger(dependency.Name()+" ", ctx.Log))
	return pip.Run(ctx.AsDependency(dependency))
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

	ctx = ctx.WithLogger(log.NewDefaultPrefixLogger(configPipeline.Name+" ", ctx.Log))
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
