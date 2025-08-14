package dependency

import (
	"bytes"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Manager can update, build, deploy and purge dependencies.
type Manager interface {
	// ResolveAll resolves all dependencies and returns them
	ResolveAll(ctx devspacecontext.Context, options ResolveOptions) ([]types.Dependency, error)
}

type manager struct {
	resolver ResolverInterface
}

// NewManager creates a new instance of the interface Manager
func NewManager(ctx devspacecontext.Context, configOptions *loader.ConfigOptions) Manager {
	return &manager{
		resolver: NewResolver(ctx, configOptions),
	}
}

func NewManagerWithParser(ctx devspacecontext.Context, configOptions *loader.ConfigOptions, parser loader.Parser) Manager {
	return &manager{
		resolver: NewResolver(ctx, configOptions).WithParser(parser),
	}
}

type ResolveOptions struct {
	SkipDependencies []string
	Dependencies     []string
}

func (m *manager) ResolveAll(ctx devspacecontext.Context, options ResolveOptions) ([]types.Dependency, error) {
	dependencies, err := m.handleDependencies(ctx, options, "Resolve", func(ctx devspacecontext.Context, dependency *Dependency) error {
		return nil
	})
	if err != nil {
		return nil, err
	}

	return dependencies, nil
}

// BuildOptions has all options for building all dependencies
type BuildOptions struct {
	BuildOptions build.Options

	SkipDependencies []string
	Dependencies     []string
	Verbose          bool
}

func (m *manager) handleDependencies(ctx devspacecontext.Context, options ResolveOptions, actionName string, action func(ctx devspacecontext.Context, dependency *Dependency) error) ([]types.Dependency, error) {
	if ctx.Config() == nil || ctx.Config().Config() == nil || len(ctx.Config().Config().Dependencies) == 0 {
		return nil, nil
	}

	hooksErr := hook.ExecuteHooks(ctx, nil, "before:"+strings.ToLower(actionName)+"Dependencies")
	if hooksErr != nil {
		return nil, hooksErr
	}

	// Resolve all dependencies
	dependencies, err := m.resolver.Resolve(ctx, options)
	if err != nil {
		return nil, errors.Wrap(err, "resolve dependencies")
	}

	executedDependencies, err := m.executeDependenciesRecursive(ctx, "", dependencies, options, actionName, action, map[string]bool{})
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

func (m *manager) executeDependenciesRecursive(ctx devspacecontext.Context, base string, dependencies []types.Dependency, options ResolveOptions, actionName string, action func(ctx devspacecontext.Context, dependency *Dependency) error, executedDependenciesIDs map[string]bool) ([]types.Dependency, error) {
	// Execute all dependencies
	i := 0
	executedDependencies := []types.Dependency{}
	for i >= 0 && i < len(dependencies) {
		var (
			dependency = dependencies[i]
			buff       = &bytes.Buffer{}
		)

		// Increase counter
		i++

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

			_, err := m.executeDependenciesRecursive(dependencyCtx, dependencyName, dependency.Children(), options, actionName, action, executedDependenciesIDs)
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
		if !foundDependency(dependencyName, options.Dependencies) {
			continue
		} else if skipDependency(dependencyName, options.SkipDependencies) {
			ctx.Log().Infof("Skip dependency %s", dependencyName)
			continue
		}

		// If not verbose log to a stream
		dependencyCtx = dependencyCtx.WithLogger(log.NewStreamLogger(buff, buff, logrus.InfoLevel))
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
