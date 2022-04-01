package dependency

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/graph"
	"github.com/loft-sh/devspace/pkg/util/kubeconfig"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/util"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/pkg/errors"
)

// ResolverInterface defines the resolver interface that takes dependency configs and resolves them
type ResolverInterface interface {
	Resolve(ctx *devspacecontext.Context) ([]types.Dependency, error)
}

// Resolver implements the resolver interface
type resolver struct {
	DependencyGraph *graph.Graph

	BaseCache  localcache.Cache
	BaseConfig *latest.Config

	ConfigOptions *loader.ConfigOptions
}

// NewResolver creates a new resolver for resolving dependencies
func NewResolver(ctx *devspacecontext.Context, configOptions *loader.ConfigOptions) ResolverInterface {
	return &resolver{
		DependencyGraph: graph.NewGraph(graph.NewNode(ctx.Config.Config().Name, &Dependency{name: ctx.Config.Config().Name, root: true})),

		BaseConfig: ctx.Config.Config(),
		BaseCache:  ctx.Config.LocalCache(),

		ConfigOptions: configOptions,
	}
}

// Resolve implements interface
func (r *resolver) Resolve(ctx *devspacecontext.Context) ([]types.Dependency, error) {
	currentWorkingDirectory, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "get current working directory")
	}

	// r.DependencyGraph.Root.ID == name here
	err = r.resolveRecursive(ctx, currentWorkingDirectory, r.DependencyGraph.Root.ID, nil, transformMap(r.BaseConfig.Dependencies))
	if err != nil {
		if _, ok := err.(*graph.CyclicError); ok {
			return nil, err
		}

		return nil, err
	}

	// Save local cache
	err = r.BaseCache.Save()
	if err != nil {
		return nil, err
	}

	// get direct children
	children := []types.Dependency{}
	for _, v := range r.DependencyGraph.Root.Childs {
		children = append(children, v.Data.(*Dependency))
	}

	return children, nil
}

func (r *resolver) resolveRecursive(ctx *devspacecontext.Context, basePath, parentConfigName string, currentDependency *Dependency, dependencies []*latest.DependencyConfig) error {
	if currentDependency != nil {
		currentDependency.children = []types.Dependency{}
	}
	for _, dependencyConfig := range dependencies {
		dependencyConfigPath, err := util.DownloadDependency(ctx.Context, basePath, dependencyConfig.Source, ctx.Log)
		if err != nil {
			return err
		}

		// Try to insert new edge
		var (
			child *Dependency
		)
		if n, ok := r.DependencyGraph.Nodes[dependencyConfig.Name]; ok {
			child = n.Data.(*Dependency)
			if child.Config().Path() != dependencyConfigPath {
				ctx.Log.Warnf("Seems like you have multiple dependencies with name %s, but they use different source settings (%s != %s). This can lead to unexpected results and you should make sure that the devspace.yaml name is unique across your dependencies or that you use the dependencies.overrideName option", child.name, child.Config().Path(), dependencyConfigPath)
			}

			err := r.DependencyGraph.AddEdge(parentConfigName, dependencyConfig.Name)
			if err != nil {
				if _, ok := err.(*graph.CyclicError); !ok {
					return err
				}

				ctx.Log.Debugf(err.Error())
			}
		} else {
			child, err = r.resolveDependency(ctx, dependencyConfigPath, dependencyConfig.Name, dependencyConfig)
			if err != nil {
				return err
			}

			_, err = r.DependencyGraph.InsertNodeAt(parentConfigName, dependencyConfig.Name, child)
			if err != nil {
				return errors.Wrap(err, "insert node")
			}

			// load dependencies from dependency
			if !dependencyConfig.IgnoreDependencies && child.localConfig.Config().Dependencies != nil && len(child.localConfig.Config().Dependencies) > 0 {
				err = r.resolveRecursive(ctx, child.absolutePath, dependencyConfig.Name, child, transformMap(child.localConfig.Config().Dependencies))
				if err != nil {
					return err
				}
			}
		}

		// add child
		if currentDependency != nil && currentDependency.children != nil && child != nil {
			currentDependency.children = append(currentDependency.children, child)
		}
	}

	return nil
}

func transformMap(depMap map[string]*latest.DependencyConfig) []*latest.DependencyConfig {
	dependencies := []*latest.DependencyConfig{}
	for _, dep := range depMap {
		dependencies = append(dependencies, dep)
	}
	sort.SliceStable(dependencies, func(i, j int) bool {
		return dependencies[i].Name < dependencies[j].Name
	})
	return dependencies
}

func (r *resolver) resolveDependency(ctx *devspacecontext.Context, dependencyConfigPath, dependencyName string, dependency *latest.DependencyConfig) (*Dependency, error) {
	// clone config options
	cloned, err := r.ConfigOptions.Clone()
	if err != nil {
		return nil, errors.Wrap(err, "clone config options")
	}

	// set dependency profile
	cloned.OverrideName = dependency.Name
	cloned.Profiles = []string{}
	cloned.Profiles = append(cloned.Profiles, dependency.Profiles...)
	cloned.DisableProfileActivation = dependency.DisableProfileActivation || r.ConfigOptions.DisableProfileActivation

	// load config
	if cloned.Vars == nil {
		cloned.Vars = []string{}
	}

	if dependency.OverwriteVars {
		for k, v := range ctx.Config.Variables() {
			cloned.Vars = append(cloned.Vars, strings.TrimSpace(k)+"="+strings.TrimSpace(fmt.Sprintf("%v", v)))
		}
	}
	for k, v := range dependency.Vars {
		cloned.Vars = append(cloned.Vars, strings.TrimSpace(k)+"="+strings.TrimSpace(v))
	}

	// recreate client if necessary
	client := ctx.KubeClient
	if dependency.Namespace != "" {
		if ctx.KubeClient == nil {
			client, err = kubectl.NewClientFromContext("", dependency.Namespace, false, kubeconfig.NewLoader())
		} else {
			client, err = kubectl.NewClientFromContext(client.CurrentContext(), dependency.Namespace, false, ctx.KubeClient.KubeConfigLoader())
		}
		if err != nil {
			return nil, errors.Wrap(err, "create new client")
		}
	}

	// load the dependency config
	var dConfigWrapper config.Config
	err = executeInDirectory(filepath.Dir(dependencyConfigPath), func() error {
		configLoader, err := loader.NewConfigLoader(dependencyConfigPath)
		if err != nil {
			return err
		}

		dConfigWrapper, err = configLoader.Load(ctx.Context, client, cloned, ctx.Log)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("loading config for dependency %s", dependencyName))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// set parsed variables in parent config
	if dependency.OverwriteVars {
		baseVars := ctx.Config.Variables()
		for k, v := range dConfigWrapper.Variables() {
			_, ok := baseVars[k]
			if !ok {
				baseVars[k] = v
			}
		}
	}

	// Create registry client for pull secrets
	return &Dependency{
		name:         dependencyName,
		absolutePath: filepath.Dir(dependencyConfigPath),
		localConfig:  dConfigWrapper,

		dependencyConfig: dependency,
		dependencyCache:  r.BaseCache,

		kubeClient: client,
	}, nil
}

func executeInDirectory(dir string, fn func() error) error {
	oldWorkingDirectory, err := os.Getwd()
	if err != nil {
		return err
	}

	err = os.Chdir(dir)
	if err != nil {
		return err
	}

	defer func() { _ = os.Chdir(oldWorkingDirectory) }()
	return fn()
}
