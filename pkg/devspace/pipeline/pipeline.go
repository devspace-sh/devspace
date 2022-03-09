package pipeline

import (
	"fmt"
	"strings"
	"sync"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/registry"
	types2 "github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/devpod"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/randutil"
	"github.com/loft-sh/devspace/pkg/util/stringutil"
	"github.com/pkg/errors"
)

func NewPipeline(name string, devPodManager devpod.Manager, dependencyRegistry registry.DependencyRegistry, config *latest.Pipeline, options types.Options) types.Pipeline {
	pip := &pipeline{
		name:               name,
		devPodManager:      devPodManager,
		dependencyRegistry: dependencyRegistry,
		dependencies:       map[string]types.Pipeline{},
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

	dependencies map[string]types.Pipeline
	parent       types.Pipeline

	main *Job
	jobs map[string]*Job
}

func (p *pipeline) Parent() types.Pipeline {
	return p.parent
}

func (p *pipeline) Close() error {
	err := p.main.Stop()
	if err != nil {
		return err
	}

	p.devPodManager.Close()
	return nil
}

func (p *pipeline) Options() types.Options {
	return p.options
}

// WaitDev waits for the dev pod managers to complete.
// This essentially waits until all dev pods are closed.
func (p *pipeline) WaitDev() {
	children := []types.Pipeline{}
	p.m.Lock()
	for _, v := range p.dependencies {
		children = append(children, v)
	}
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

func (p *pipeline) Dependencies() map[string]types.Pipeline {
	p.m.Lock()
	defer p.m.Unlock()

	children := map[string]types.Pipeline{}
	for k, v := range p.dependencies {
		children[k] = v
	}
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

func (p *pipeline) startNewDependency(ctx *devspacecontext.Context, dependency types2.Dependency, options types.DependencyOptions) error {
	// find the dependency pipeline to execute
	executePipeline := options.Pipeline
	if executePipeline == "" {
		if dependency.DependencyConfig().Pipeline != "" {
			executePipeline = dependency.DependencyConfig().Pipeline
		} else {
			executePipeline = "deploy"
		}
	}

	// find pipeline
	var (
		pipelineConfig *latest.Pipeline
		err            error
	)
	if dependency.Config().Config().Pipelines == nil || dependency.Config().Config().Pipelines[executePipeline] == nil {
		pipelineConfig, err = GetDefaultPipeline(executePipeline)
		if err != nil {
			return err
		}
	} else {
		pipelineConfig = dependency.Config().Config().Pipelines[executePipeline]
	}

	dependencyDevPodManager := devpod.NewManager(p.devPodManager.Context())
	pip := NewPipeline(dependency.Name(), dependencyDevPodManager, p.dependencyRegistry, pipelineConfig, p.options)
	pip.(*pipeline).parent = p

	p.m.Lock()
	p.dependencies[dependency.Name()] = pip
	p.m.Unlock()

	if streamLogger, ok := ctx.Log.(*log.StreamLogger); !ok || streamLogger.GetFormat() != log.RawFormat {
		ctx = ctx.WithLogger(log.NewDefaultPrefixLogger(dependency.Name()+" ", ctx.Log))
	}
	return pip.Run(ctx.AsDependency(dependency))
}

func (p *pipeline) StartNewPipelines(ctx *devspacecontext.Context, pipelines []*latest.Pipeline, options types.PipelineOptions) error {
	if options.Background {
		for _, configPipeline := range pipelines {
			go func(configPipeline *latest.Pipeline) {
				err := p.startNewPipeline(ctx, configPipeline, randutil.GenerateRandomString(5), options)
				if err != nil {
					ctx.Log.Errorf("starting pipeline %s has failed: %v", configPipeline.Name, err)
				}
			}(configPipeline)
		}
		return nil
	} else if options.Sequential {
		for _, configPipeline := range pipelines {
			err := p.startNewPipeline(ctx, configPipeline, randutil.GenerateRandomString(5), options)
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
					return p.startNewPipeline(ctx, configPipeline, randutil.GenerateRandomString(5), options)
				})
			}(configPipeline)
		}
		return nil
	})

	return t.Wait()
}

func (p *pipeline) startNewPipeline(ctx *devspacecontext.Context, configPipeline *latest.Pipeline, id string, options types.PipelineOptions) error {
	if configPipeline.Name == p.name {
		return fmt.Errorf("pipeline %s is already running", configPipeline.Name)
	}

	// parse env
	envMap := map[string]string{}
	for _, s := range options.Env {
		if s == "" {
			continue
		}

		splitted := strings.Split(s, "=")
		if len(splitted) <= 1 {
			return fmt.Errorf("invalid environment variable format. Has to be KEY=VALUE")
		}

		envMap[splitted[0]] = strings.Join(splitted[1:], "=")
	}

	// exchange job if it's not alive anymore
	id, j, err := p.createJob(configPipeline, envMap)
	if err != nil {
		return err
	}
	defer p.removeJob(j, id)

	err = p.executeJob(ctx, j)
	if err != nil {
		return err
	}

	return nil
}

func (p *pipeline) createJob(configPipeline *latest.Pipeline, envMap map[string]string) (id string, job *Job, err error) {
	p.m.Lock()
	defer p.m.Unlock()

	j, ok := p.jobs[id]
	if ok && !j.Terminated() {
		return "", nil, fmt.Errorf("pipeline %s is already running", id)
	}

	j = &Job{
		Pipeline: p,
		Config:   configPipeline,
		ExtraEnv: envMap,
	}
	p.jobs[id] = j
	return id, j, nil
}

func (p *pipeline) removeJob(j *Job, id string) {
	p.m.Lock()
	defer p.m.Unlock()

	nj, ok := p.jobs[id]
	if !ok {
		return
	} else if nj == j {
		delete(p.jobs, id)
	}
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
