package pipeline

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/context/values"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/tomb"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"mvdan.cc/sh/v3/expand"

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
		t:        &tomb.Tomb{},
	}
	return pip
}

type pipeline struct {
	m sync.Mutex

	excluded    bool
	excludedErr error

	options types.Options

	// name is the devspace config name, not the name of the pipeline
	name string

	devPodManager      devpod.Manager
	dependencyRegistry registry.DependencyRegistry

	dependencies map[string]types.Pipeline
	parent       types.Pipeline

	main *Job
	jobs map[string]*Job
}

func (p *pipeline) Done() <-chan struct{} {
	return p.main.Done()
}

func (p *pipeline) Exclude(ctx devspacecontext.Context) error {
	// get parent
	if p.Parent() != nil {
		parent := p.Parent()
		for parent.Parent() != nil {
			parent = parent.Parent()
		}

		return parent.Exclude(ctx)
	}

	// make sure we are locked
	p.m.Lock()
	defer p.m.Unlock()

	if p.excluded {
		return p.excludedErr
	}

	p.excluded = true

	// create namespace if necessary
	if ctx.KubeClient() != nil {
		p.excludedErr = kubectl.EnsureNamespace(ctx.Context(), ctx.KubeClient(), ctx.KubeClient().Namespace(), ctx.Log())
		if p.excludedErr != nil {
			p.excludedErr = errors.Errorf("unable to create namespace: %v", p.excludedErr)
			return p.excludedErr
		}
	}

	// exclude ourselves
	var couldExclude map[string]registry.LockType
	couldExclude, p.excludedErr = p.dependencyRegistry.TryLockDependencies(ctx, p.name, []string{p.name}, true)
	if p.excludedErr != nil {
		return p.excludedErr
	} else if couldExclude[p.name] != registry.Locked && ctx.KubeClient() != nil {
		return fmt.Errorf("couldn't execute '%s', because there is another DevSpace session for the project (%s) already running inside this namespace\n\n%s\n ", strings.Join(os.Args, " "), p.name, `You may want to use one of these commands instead:
- devspace enter: opens a terminal session for a container
- devspace attach: attaches to the PID 1 process (entrypoint) of a container
- devspace logs: streams the logs of a container
- devspace sync: syncs files between your local filesyste and a container's filesystem
- devspace ui: starts the DevSpace localhost UI`)
	}
	ctx.Log().Debugf("Marked project excluded: %v", p.name)
	return nil
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
func (p *pipeline) WaitDev() error {
	children := []types.Pipeline{}
	p.m.Lock()
	for _, v := range p.dependencies {
		children = append(children, v)
	}
	p.m.Unlock()

	// wait for children first
	errors := []error{}
	for _, child := range children {
		err := child.WaitDev()
		if err != nil {
			errors = append(errors, err)
		}
	}

	// wait for dev pods to finish
	err := p.devPodManager.Wait()
	if err != nil {
		errors = append(errors, err)
	}

	return utilerrors.NewAggregate(errors)
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

func (p *pipeline) Run(ctx devspacecontext.Context, args []string) error {
	return p.executeJob(ctx, p.main, args, ctx.Environ())
}

func (p *pipeline) StartNewDependencies(ctx devspacecontext.Context, dependencies []types2.Dependency, options types.DependencyOptions) error {
	// mark all commands from here that they are running within a dependency
	ctx = ctx.WithContext(values.WithDependency(ctx.Context(), true))
	dependencyNames := []string{}
	for _, dependency := range dependencies {
		dependencyNames = append(dependencyNames, dependency.Name())
	}

	lockedDependencies, err := p.dependencyRegistry.TryLockDependencies(ctx, p.name, dependencyNames, false)
	if err != nil {
		return errors.Wrap(err, "check if dependencies can be deployed")
	}

	deployDependencies := []types2.Dependency{}
	for _, dependency := range dependencies {
		if len(options.Only) > 0 && !stringutil.Contains(options.Only, dependency.Name()) {
			ctx.Log().Debugf("Skipping dependency %s because it was excluded", dependency.Name())
			continue
		} else if stringutil.Contains(options.Exclude, dependency.Name()) {
			ctx.Log().Debugf("Skipping dependency %s because it was excluded", dependency.Name())
			continue
		} else if lockedDependencies[dependency.Name()] != registry.Locked {
			// search for dependency pipeline and wait
			if lockedDependencies[dependency.Name()] == registry.InUse {
				ctx.Log().Infof("Skipping dependency %s as it was already deployed", dependency.Name())
				waitForDependency(ctx.Context(), p, dependency.Name(), ctx.Log())
			} else if lockedDependencies[dependency.Name()] == registry.InUseCyclic {
				ctx.Log().Infof("Skipping dependency %s as it was already deployed (cyclic)", dependency.Name())
			} else if lockedDependencies[dependency.Name()] == registry.InUseByOtherInstance {
				ctx.Log().Infof("Skipping dependency %s as it is currently in use by another DevSpace instance in the same namespace", dependency.Name())
			}
			continue
		}

		err := ensureNamespace(ctx, dependency.DependencyConfig().Namespace)
		if err != nil {
			return errors.Wrapf(err, "cannot run dependency %s", dependency.Name())
		}
		deployDependencies = append(deployDependencies, dependency)
	}

	// Start sequentially
	if options.Sequential {
		ctx.Log().Debug("Deploying dependencies sequentially")
		for _, dependency := range deployDependencies {
			err := p.startNewDependency(ctx, dependency, options)
			if err != nil {
				return errors.Wrapf(err, "run dependency %s", dependency.Name())
			} else {
				ctx.Log().Debugf("Dependency '%s' deployed", dependency.Name())
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

func ensureNamespace(ctx devspacecontext.Context, namespace string) error {
	// If localregistry namespace is the same as devspace, we don't have
	// anything to do.
	if namespace == ctx.KubeClient().Namespace() {
		ctx.Log().Debugf("Namespace %s is the default Devspace namespace", namespace)
		return nil
	}

	return kubectl.EnsureNamespace(ctx.Context(), ctx.KubeClient(), namespace, ctx.Log())
}

func waitForDependency(ctx context.Context, start types.Pipeline, dependencyName string, log log.Logger) {
	// get top level pipeline
	for start.Parent() != nil {
		start = start.Parent()
	}

	// try to find the dependency
	var pipeline types.Pipeline
	err := wait.PollImmediateWithContext(ctx, time.Millisecond*10, time.Second, func(_ context.Context) (bool, error) {
		pipeline = findDependencies(start, dependencyName)
		return pipeline != nil, nil
	})
	if err != nil {
		log.Debugf("error finding dependency: %v", err)
	}

	// wait for dependency
	if pipeline != nil {
		log.Infof("Waiting for dependency '%s' to finish...", dependencyName)
		select {
		case <-pipeline.Done():
		case <-ctx.Done():
		}
	}
}

func findDependencies(start types.Pipeline, dependencyName string) types.Pipeline {
	for key, pipe := range start.Dependencies() {
		if key == dependencyName {
			return pipe
		}

		found := findDependencies(pipe, dependencyName)
		if found != nil {
			return found
		}
	}

	return nil
}

func (p *pipeline) StartNewPipelines(ctx devspacecontext.Context, pipelines []*latest.Pipeline, options types.PipelineOptions) error {
	if options.Background {
		for _, configPipeline := range pipelines {
			go func(configPipeline *latest.Pipeline) {
				err := p.startNewPipeline(ctx, configPipeline, randutil.GenerateRandomString(5), options)
				if err != nil {
					ctx.Log().Errorf("starting pipeline %s has failed: %v", configPipeline.Name, err)
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

func (p *pipeline) startNewPipeline(ctx devspacecontext.Context, configPipeline *latest.Pipeline, id string, options types.PipelineOptions) error {
	ctx, err := applyFlags(ctx, configPipeline, options.SetFlag)
	if err != nil {
		return err
	}

	// exchange job if it's not alive anymore
	j, err := p.createJob(configPipeline, id)
	if err != nil {
		return err
	}
	defer p.removeJob(j, id)

	err = p.executeJob(ctx, j, nil, options.Environ)
	if err != nil {
		return err
	}

	return nil
}

func (p *pipeline) startNewDependency(ctx devspacecontext.Context, dependency types2.Dependency, options types.DependencyOptions) error {
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
		pipelineConfig, err = types.GetDefaultPipeline(executePipeline)
		if err != nil {
			return err
		}
	} else {
		pipelineConfig = dependency.Config().Config().Pipelines[executePipeline]
	}

	ctx, err = applyFlags(ctx, pipelineConfig, options.SetFlag)
	if err != nil {
		return err
	}

	devCtx, _ := values.DevContextFrom(ctx.Context())
	devCtxCancel, cancelDevCtx := context.WithCancel(devCtx)
	ctx = ctx.WithContext(values.WithDevContext(ctx.Context(), devCtxCancel))
	dependencyDevPodManager := devpod.NewManager(cancelDevCtx)
	pip := NewPipeline(dependency.Name(), dependencyDevPodManager, p.dependencyRegistry, pipelineConfig, p.options)
	pip.(*pipeline).parent = p

	p.m.Lock()
	p.dependencies[dependency.Name()] = pip
	p.m.Unlock()

	if streamLogger, ok := ctx.Log().(*log.StreamLogger); !ok || streamLogger.GetFormat() != log.RawFormat {
		ctx = ctx.WithLogger(ctx.Log().WithPrefix(dependency.Name() + " "))
	}
	return pip.Run(ctx.AsDependency(dependency), nil)
}

func applyFlags(ctx devspacecontext.Context, pipeline *latest.Pipeline, setFlags []string) (devspacecontext.Context, error) {
	newFlags := map[string]string{}
	for _, flag := range pipeline.Flags {
		newFlags[flag.Name] = fmt.Sprintf("%v", flag.Default)
	}
	for _, v := range setFlags {
		splitted := strings.Split(v, "=")
		if len(splitted) <= 1 {
			return nil, fmt.Errorf("error parsing flag %s: expected format flag=value", v)
		}

		newFlags[splitted[0]] = strings.Join(splitted[1:], "=")
	}

	return ctx.WithContext(values.WithFlagsMap(ctx.Context(), newFlags)), nil
}

func (p *pipeline) createJob(configPipeline *latest.Pipeline, id string) (job *Job, err error) {
	p.m.Lock()
	defer p.m.Unlock()

	j, ok := p.jobs[id]
	if ok && !j.Terminated() {
		return nil, fmt.Errorf("pipeline %s is already running", id)
	}

	j = &Job{
		Pipeline: p,
		Config:   configPipeline,
		t:        &tomb.Tomb{},
	}
	p.jobs[id] = j
	return j, nil
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

func (p *pipeline) executeJob(ctx devspacecontext.Context, j *Job, args []string, environ expand.Environ) error {
	// don't start jobs on a cancelled context
	if ctx.IsDone() {
		return nil
	}

	err := j.Run(ctx, args, environ)
	if err != nil {
		return err
	}

	return nil
}
