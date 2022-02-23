package dependency

import (
	"bytes"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/exit"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"mvdan.cc/sh/v3/interp"
	"strings"
)

// Manager can update, build, deploy and purge dependencies.
type Manager interface {
	// UpdateAll updates all dependencies
	UpdateAll() error

	// BuildAll builds all dependencies
	BuildAll(options BuildOptions) ([]types.Dependency, error)

	// DeployAll deploys all dependencies and returns them
	DeployAll(options DeployOptions) ([]types.Dependency, error)

	// ResolveAll resolves all dependencies and returns them
	ResolveAll(options ResolveOptions) ([]types.Dependency, error)

	// PurgeAll purges all dependencies
	PurgeAll(options PurgeOptions) ([]types.Dependency, error)

	// RenderAll renders all dependencies
	RenderAll(options RenderOptions) ([]types.Dependency, error)
}

type manager struct {
	config   config.Config
	log      log.Logger
	resolver ResolverInterface
	client   kubectl.Client
}

// NewManager creates a new instance of the interface Manager
func NewManager(config config.Config, client kubectl.Client, configOptions *loader.ConfigOptions, logger log.Logger) Manager {
	return &manager{
		config:   config,
		log:      logger,
		resolver: NewResolver(config, client, configOptions, logger),
		client:   client,
	}
}

// UpdateAll will update all dependencies if there are any
func (m *manager) UpdateAll() error {
	if m.config == nil || m.config.Config() == nil || len(m.config.Config().Dependencies) == 0 {
		return nil
	}

	m.log.StartWait("Update dependencies")
	defer m.log.StopWait()

	// Resolve all dependencies
	_, err := m.resolver.Resolve(true)
	if err != nil {
		if _, ok := err.(*cyclicError); ok {
			return errors.Errorf("%v.\n To allow cyclic dependencies run with the '%s' flag", err, ansi.Color("--allow-cyclic", "white+b"))
		}

		return err
	}

	return nil
}

type ResolveOptions struct {
	SkipDependencies   []string
	Dependencies       []string
	UpdateDependencies bool
	Silent             bool
	Verbose            bool
}

func (m *manager) ResolveAll(options ResolveOptions) ([]types.Dependency, error) {
	dependencies, err := m.handleDependencies(options.SkipDependencies, options.Dependencies, false, options.UpdateDependencies, options.Silent, options.Verbose, "Resolve", func(dependency *Dependency, log log.Logger) error {
		return nil
	})
	if err != nil {
		return nil, err
	}

	return dependencies, nil
}

// ExecuteCommand executes a given command from the available commands
func ExecuteCommand(commands []*latest.CommandConfig, cmd string, args []string, stdout io.Writer, stderr io.Writer) error {
	err := command.ExecuteCommand(commands, cmd, args, stdout, stderr)
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

	SkipDependencies        []string
	Dependencies            []string
	UpdateDependencies      bool
	ForceDeployDependencies bool
	Verbose                 bool
}

// BuildAll will build all dependencies if there are any
func (m *manager) BuildAll(options BuildOptions) ([]types.Dependency, error) {
	return m.handleDependencies(options.SkipDependencies, options.Dependencies, false, options.UpdateDependencies, false, options.Verbose, "Build", func(dependency *Dependency, log log.Logger) error {
		return dependency.Build(options.ForceDeployDependencies, &options.BuildOptions, log)
	})
}

// DeployOptions has all options for deploying all dependencies
type DeployOptions struct {
	BuildOptions build.Options

	SkipDependencies        []string
	Dependencies            []string
	UpdateDependencies      bool
	ForceDeployDependencies bool
	SkipBuild               bool
	SkipDeploy              bool
	ForceDeploy             bool
	Verbose                 bool
}

// DeployAll will deploy all dependencies if there are any
func (m *manager) DeployAll(options DeployOptions) ([]types.Dependency, error) {
	dependencies, err := m.handleDependencies(options.SkipDependencies, options.Dependencies, false, options.UpdateDependencies, false, options.Verbose, "Deploy", func(dependency *Dependency, log log.Logger) error {
		return dependency.Deploy(options.ForceDeployDependencies, options.SkipBuild, options.SkipDeploy, options.ForceDeploy, &options.BuildOptions, log)
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
func (m *manager) PurgeAll(options PurgeOptions) ([]types.Dependency, error) {
	return m.handleDependencies(options.SkipDependencies, options.Dependencies, true, false, false, options.Verbose, "Purge", func(dependency *Dependency, log log.Logger) error {
		return dependency.Purge(log)
	})
}

// RenderOptions has all options for rendering all dependencies
type RenderOptions struct {
	SkipDependencies   []string
	Dependencies       []string
	Verbose            bool
	UpdateDependencies bool
	SkipBuild          bool
	Writer             io.Writer

	BuildOptions build.Options
}

func (m *manager) RenderAll(options RenderOptions) ([]types.Dependency, error) {
	return m.handleDependencies(options.SkipDependencies, options.Dependencies, false, options.UpdateDependencies, false, options.Verbose, "Render", func(dependency *Dependency, log log.Logger) error {
		return dependency.Render(options.SkipBuild, &options.BuildOptions, options.Writer, log)
	})
}

func (m *manager) handleDependencies(skipDependencies, filterDependencies []string, reverse, updateDependencies, silent, verbose bool, actionName string, action func(dependency *Dependency, log log.Logger) error) ([]types.Dependency, error) {
	if m.config == nil || m.config.Config() == nil || len(m.config.Config().Dependencies) == 0 {
		return nil, nil
	}

	if !silent {
		m.log.Infof("Start resolving dependencies")
	}

	hooksErr := hook.ExecuteHooks(m.client, m.config, nil, nil, m.log, "before:"+strings.ToLower(actionName)+"Dependencies")
	if hooksErr != nil {
		return nil, hooksErr
	}

	// Resolve all dependencies
	dependencies, err := m.resolver.Resolve(updateDependencies)
	if err != nil {
		return nil, errors.Wrap(err, "resolve dependencies")
	}

	defer m.log.StopWait()

	if !silent {
		m.log.Donef("Resolved dependencies successfully")
	}
	if !silent && !verbose {
		m.log.Infof("To display the complete dependency execution log run with the '--verbose-dependencies' flag")
	}

	executedDependencies, err := m.executeDependenciesRecursive("", dependencies, skipDependencies, filterDependencies, reverse, silent, verbose, actionName, action, map[string]bool{})
	if err != nil {
		hooksErr := hook.ExecuteHooks(m.client, m.config, dependencies, map[string]interface{}{
			"error": err,
		}, m.log, "error:"+strings.ToLower(actionName)+"Dependencies")
		if hooksErr != nil {
			return nil, hooksErr
		}

		return nil, err
	}

	hooksErr = hook.ExecuteHooks(m.client, m.config, dependencies, nil, m.log, "after:"+strings.ToLower(actionName)+"Dependencies")
	if hooksErr != nil {
		return nil, hooksErr
	}

	return executedDependencies, nil
}

func (m *manager) executeDependenciesRecursive(base string, dependencies []types.Dependency, skipDependencies, filterDependencies []string, reverse, silent, verbose bool, actionName string, action func(dependency *Dependency, log log.Logger) error, executedDependenciesIDs map[string]bool) ([]types.Dependency, error) {
	// Execute all dependencies
	i := 0
	if reverse {
		i = len(dependencies) - 1
	}

	executedDependencies := []types.Dependency{}
	for i >= 0 && i < len(dependencies) {
		var (
			dependency       = dependencies[i]
			buff             = &bytes.Buffer{}
			dependencyLogger = m.log
		)

		// Increase / Decrease counter
		if reverse {
			i--
		} else {
			i++
		}

		// skip if dependency was executed already
		if executedDependenciesIDs[dependency.ID()] {
			executedDependencies = append(executedDependencies, dependency)
			continue
		}

		// make sure we don't execute the dependency again
		executedDependenciesIDs[dependency.ID()] = true

		// get dependency name
		dependencyName := dependency.Name()
		if base != "" {
			dependencyName = base + "." + dependencyName
		}

		// deploy the dependencies of the dependency first
		if len(dependency.Children()) > 0 {
			hooksErr := hook.ExecuteHooks(dependency.KubeClient(), dependency.Config(), dependency.Children(), nil, m.log, "before:"+strings.ToLower(actionName)+"Dependencies")
			if hooksErr != nil {
				return nil, hooksErr
			}

			_, err := m.executeDependenciesRecursive(dependencyName, dependency.Children(), skipDependencies, filterDependencies, reverse, silent, verbose, actionName, action, executedDependenciesIDs)
			if err != nil {
				hooksErr := hook.ExecuteHooks(dependency.KubeClient(), dependency.Config(), dependency.Children(), map[string]interface{}{
					"error": err,
				}, m.log, "error:"+strings.ToLower(actionName)+"Dependencies")
				if hooksErr != nil {
					return nil, hooksErr
				}

				return nil, err
			}

			hooksErr = hook.ExecuteHooks(dependency.KubeClient(), dependency.Config(), dependency.Children(), nil, m.log, "after:"+strings.ToLower(actionName)+"Dependencies")
			if hooksErr != nil {
				return nil, hooksErr
			}
		}

		// Check if we should act on this dependency
		if !foundDependency(dependencyName, filterDependencies) {
			continue
		} else if skipDependency(dependencyName, skipDependencies) {
			m.log.Infof("Skip dependency %s", dependencyName)
			continue
		}

		// execute dependency
		if !silent && !verbose {
			m.log.Infof(fmt.Sprintf("%s dependency %s...", actionName, dependencyName))
		}

		// If not verbose log to a stream
		if !verbose {
			dependencyLogger = log.NewStreamLogger(buff, logrus.InfoLevel)
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

		err := action(dependency.(*Dependency), dependencyLogger)
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
			m.log.Donef("%s dependency %s completed", actionName, dependencyName)
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
