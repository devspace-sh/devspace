package dependency

import (
	"bytes"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/command"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/exit"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"mvdan.cc/sh/v3/interp"
	"strings"
)

// Manager can update, build, deploy and purge dependencies.
type Manager interface {
	// BuildAll builds all dependencies
	BuildAll(ctx *devspacecontext.Context, options BuildOptions) ([]types.Dependency, error)

	// DeployAll deploys all dependencies and returns them
	DeployAll(ctx *devspacecontext.Context, options DeployOptions) ([]types.Dependency, error)

	// ResolveAll resolves all dependencies and returns them
	ResolveAll(ctx *devspacecontext.Context, options ResolveOptions) ([]types.Dependency, error)

	// PurgeAll purges all dependencies
	PurgeAll(ctx *devspacecontext.Context, options PurgeOptions) ([]types.Dependency, error)

	// RenderAll renders all dependencies
	RenderAll(ctx *devspacecontext.Context, options RenderOptions) ([]types.Dependency, error)
}

type manager struct {
	resolver ResolverInterface
}

// NewManager creates a new instance of the interface Manager
func NewManager(ctx *devspacecontext.Context, configOptions *loader.ConfigOptions) Manager {
	return &manager{
		resolver: NewResolver(ctx, configOptions),
	}
}

type ResolveOptions struct {
	SkipDependencies []string
	Dependencies     []string
	Silent           bool
	Verbose          bool
}

func (m *manager) ResolveAll(ctx *devspacecontext.Context, options ResolveOptions) ([]types.Dependency, error) {
	dependencies, err := m.handleDependencies(ctx, options.SkipDependencies, options.Dependencies, false, options.Silent, options.Verbose, "Resolve", func(ctx *devspacecontext.Context, dependency *Dependency) error {
		return nil
	})
	if err != nil {
		return nil, err
	}

	return dependencies, nil
}

// ExecuteCommand executes a given command from the available commands
func ExecuteCommand(commands []*latest.CommandConfig, cmd string, args []string, dir string, stdout io.Writer, stderr io.Writer) error {
	err := command.ExecuteCommand(commands, cmd, args, dir, stdout, stderr)
	if err != nil {
		if status, ok := interp.IsExitStatus(err); ok {
			return &exit.ReturnCodeError{
				ExitCode: int(status),
			}
		}

		return errors.Wrap(err, "execute command")
	}

	return nil
}

// BuildOptions has all options for building all dependencies
type BuildOptions struct {
	BuildOptions build.Options

	SkipDependencies []string
	Dependencies     []string
	Verbose          bool
}

// BuildAll will build all dependencies if there are any
func (m *manager) BuildAll(ctx *devspacecontext.Context, options BuildOptions) ([]types.Dependency, error) {
	return m.handleDependencies(ctx, options.SkipDependencies, options.Dependencies, false, false, options.Verbose, "Build", func(ctx *devspacecontext.Context, dependency *Dependency) error {
		return dependency.Build(ctx, &options.BuildOptions)
	})
}

// DeployOptions has all options for deploying all dependencies
type DeployOptions struct {
	BuildOptions build.Options

	SkipDependencies []string
	Dependencies     []string
	SkipBuild        bool
	SkipDeploy       bool
	ForceDeploy      bool
	Verbose          bool
}

// DeployAll will deploy all dependencies if there are any
func (m *manager) DeployAll(ctx *devspacecontext.Context, options DeployOptions) ([]types.Dependency, error) {
	dependencies, err := m.handleDependencies(ctx, options.SkipDependencies, options.Dependencies, false, false, options.Verbose, "Deploy", func(ctx *devspacecontext.Context, dependency *Dependency) error {
		return dependency.Deploy(ctx, options.SkipBuild, options.SkipDeploy, options.ForceDeploy, &options.BuildOptions)
	})
	if err != nil {
		return nil, err
	}

	return dependencies, nil
}

// PurgeOptions has all options for purging all dependencies
type PurgeOptions struct {
	SkipDependencies []string
	Dependencies     []string
	Verbose          bool
}

// PurgeAll purges all dependencies in reverse order
func (m *manager) PurgeAll(ctx *devspacecontext.Context, options PurgeOptions) ([]types.Dependency, error) {
	return m.handleDependencies(ctx, options.SkipDependencies, options.Dependencies, true, false, options.Verbose, "Purge", func(ctx *devspacecontext.Context, dependency *Dependency) error {
		return dependency.Purge(ctx)
	})
}

// RenderOptions has all options for rendering all dependencies
type RenderOptions struct {
	SkipDependencies []string
	Dependencies     []string
	Verbose          bool
	SkipBuild        bool
	Writer           io.Writer

	BuildOptions build.Options
}

func (m *manager) RenderAll(ctx *devspacecontext.Context, options RenderOptions) ([]types.Dependency, error) {
	return m.handleDependencies(ctx, options.SkipDependencies, options.Dependencies, false, false, options.Verbose, "Render", func(ctx *devspacecontext.Context, dependency *Dependency) error {
		return dependency.Render(ctx, options.SkipBuild, &options.BuildOptions, options.Writer)
	})
}

func (m *manager) handleDependencies(ctx *devspacecontext.Context, skipDependencies, filterDependencies []string, reverse, silent, verbose bool, actionName string, action func(ctx *devspacecontext.Context, dependency *Dependency) error) ([]types.Dependency, error) {
	if ctx.Config == nil || ctx.Config.Config() == nil || len(ctx.Config.Config().Dependencies) == 0 {
		return nil, nil
	}

	if !silent {
		ctx.Log.Infof("Start resolving dependencies")
	}

	hooksErr := hook.ExecuteHooks(ctx, nil, "before:"+strings.ToLower(actionName)+"Dependencies")
	if hooksErr != nil {
		return nil, hooksErr
	}

	// Resolve all dependencies
	dependencies, err := m.resolver.Resolve(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "resolve dependencies")
	}

	defer ctx.Log.StopWait()

	if !silent {
		ctx.Log.Donef("Resolved dependencies successfully")
	}
	if !silent && !verbose {
		ctx.Log.Infof("To display the complete dependency execution log run with the '--verbose-dependencies' flag")
	}

	executedDependencies, err := m.executeDependenciesRecursive(ctx, "", dependencies, skipDependencies, filterDependencies, reverse, silent, verbose, actionName, action, map[string]bool{})
	if err != nil {
		hooksErr := hook.ExecuteHooks(ctx, map[string]interface{}{
			"error": err,
		}, "error:"+strings.ToLower(actionName)+"Dependencies")
		if hooksErr != nil {
			return nil, hooksErr
		}

		return nil, err
	}

	hooksErr = hook.ExecuteHooks(ctx, nil, "after:"+strings.ToLower(actionName)+"Dependencies")
	if hooksErr != nil {
		return nil, hooksErr
	}

	return executedDependencies, nil
}

func (m *manager) executeDependenciesRecursive(
	ctx *devspacecontext.Context,
	base string,
	dependencies []types.Dependency,
	skipDependencies, filterDependencies []string,
	reverse, silent, verbose bool,
	actionName string,
	action func(ctx *devspacecontext.Context, dependency *Dependency) error,
	executedDependenciesIDs map[string]bool,
) ([]types.Dependency, error) {
	// Execute all dependencies
	i := 0
	if reverse {
		i = len(dependencies) - 1
	}

	executedDependencies := []types.Dependency{}
	for i >= 0 && i < len(dependencies) {
		var (
			dependency = dependencies[i]
			buff       = &bytes.Buffer{}
		)

		// Increase / Decrease counter
		if reverse {
			i--
		} else {
			i++
		}

		// skip if dependency was executed already
		if executedDependenciesIDs[dependency.Name()] {
			executedDependencies = append(executedDependencies, dependency)
			continue
		}

		// make sure we don't execute the dependency again
		executedDependenciesIDs[dependency.Name()] = true

		// get dependency name
		dependencyName := dependency.Name()
		if base != "" {
			dependencyName = base + "." + dependencyName
		}

		// deploy the dependencies of the dependency first
		dependencyCtx := ctx.AsDependency(dependency)
		if len(dependency.Children()) > 0 {
			hooksErr := hook.ExecuteHooks(dependencyCtx, nil, "before:"+strings.ToLower(actionName)+"Dependencies")
			if hooksErr != nil {
				return nil, hooksErr
			}

			_, err := m.executeDependenciesRecursive(dependencyCtx, dependencyName, dependency.Children(), skipDependencies, filterDependencies, reverse, silent, verbose, actionName, action, executedDependenciesIDs)
			if err != nil {
				hooksErr := hook.ExecuteHooks(dependencyCtx, map[string]interface{}{
					"error": err,
				}, "error:"+strings.ToLower(actionName)+"Dependencies")
				if hooksErr != nil {
					return nil, hooksErr
				}

				return nil, err
			}

			hooksErr = hook.ExecuteHooks(dependencyCtx, nil, "after:"+strings.ToLower(actionName)+"Dependencies")
			if hooksErr != nil {
				return nil, hooksErr
			}
		}

		// Check if we should act on this dependency
		if !foundDependency(dependencyName, filterDependencies) {
			continue
		} else if skipDependency(dependencyName, skipDependencies) {
			ctx.Log.Infof("Skip dependency %s", dependencyName)
			continue
		}

		// execute dependency
		if !silent && !verbose {
			ctx.Log.Infof(fmt.Sprintf("%s dependency %s...", actionName, dependencyName))
		}

		// If not verbose log to a stream
		if !verbose {
			dependencyCtx = dependencyCtx.WithLogger(log.NewStreamLogger(buff, logrus.InfoLevel))
		}

		if dependency.Config() != nil {
			pluginErr := plugin.ExecutePluginHookWithContext(map[string]interface{}{
				"dependency_name":        dependency.Name(),
				"dependency_config":      dependency.Config().Config(),
				"dependency_config_path": dependency.Config().Path(),
			}, hook.EventsForSingle("before:"+strings.ToLower(actionName)+"Dependency", dependency.Name()).With("dependencies.before"+actionName)...)
			if pluginErr != nil {
				return nil, pluginErr
			}
		}

		err := action(dependencyCtx, dependency.(*Dependency))
		if err != nil {
			if dependency.Config() != nil {
				pluginErr := plugin.ExecutePluginHookWithContext(map[string]interface{}{
					"dependency_name":        dependency.Name(),
					"dependency_config":      dependency.Config().Config(),
					"dependency_config_path": dependency.Config().Path(),
				}, hook.EventsForSingle("error:"+strings.ToLower(actionName)+"Dependency", dependency.Name()).With("dependencies.error"+actionName)...)
				if pluginErr != nil {
					return nil, pluginErr
				}
			}

			return nil, errors.Wrapf(err, "%s dependency %s error %s", actionName, dependency.Name(), buff.String())
		}

		if dependency.Config() != nil {
			pluginErr := plugin.ExecutePluginHookWithContext(map[string]interface{}{
				"dependency_name":        dependency.Name(),
				"dependency_config":      dependency.Config().Config(),
				"dependency_config_path": dependency.Config().Path(),
			}, hook.EventsForSingle("after:"+strings.ToLower(actionName)+"Dependency", dependency.Name()).With("dependencies.after"+actionName)...)
			if pluginErr != nil {
				return nil, pluginErr
			}
		}

		executedDependencies = append(executedDependencies, dependency)
		if !silent {
			ctx.Log.Donef("%s dependency %s completed", actionName, dependencyName)
		}
	}

	return executedDependencies, nil
}

func GetDependencyByPath(dependencies []types.Dependency, path string) types.Dependency {
	splitted := strings.Split(path, ".")

	var retDependency types.Dependency
	searchDependencies := dependencies
	for _, segment := range splitted {
		var nextDependency types.Dependency
		for _, dependency := range searchDependencies {
			if dependency.Name() == segment {
				nextDependency = dependency
				break
			}
		}

		// not found, exit here
		if nextDependency == nil {
			return nil
		}

		searchDependencies = nextDependency.Children()
		retDependency = nextDependency
	}

	return retDependency
}
