package dependency

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/pullsecrets"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/util"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/git"
	"github.com/loft-sh/devspace/pkg/util/kubeconfig"
	"github.com/loft-sh/devspace/pkg/util/log"

	"github.com/pkg/errors"
)

// ResolverInterface defines the resolver interface that takes dependency configs and resolves them
type ResolverInterface interface {
	Resolve(update bool) ([]types.Dependency, error)
}

// Resolver implements the resolver interface
type resolver struct {
	RootID          string
	DependencyGraph *graph

	BasePath   string
	BaseCache  *generated.Config
	BaseConfig *latest.Config
	BaseVars   map[string]interface{}

	ConfigOptions *loader.ConfigOptions

	kubeLoader     kubeconfig.Loader
	client         kubectl.Client
	generatedSaver generated.ConfigLoader
	log            log.Logger
}

// NewResolver creates a new resolver for resolving dependencies
func NewResolver(baseConfig config.Config, client kubectl.Client, configOptions *loader.ConfigOptions, log log.Logger) ResolverInterface {
	var id string

	var kubeLoader kubeconfig.Loader
	if client == nil {
		kubeLoader = kubeconfig.NewLoader()
	} else {
		kubeLoader = client.KubeConfigLoader()
	}

	basePath, err := filepath.Abs(".")
	if err != nil {
		panic(err)
	}
	remote, err := git.GetRemote(basePath)
	if err == nil {
		id = remote
	} else {
		id = basePath
	}

	return &resolver{
		RootID:          id,
		DependencyGraph: newGraph(newNode(id, nil)),

		BaseConfig: baseConfig.Config(),
		BaseCache:  baseConfig.Generated(),
		BaseVars:   baseConfig.Variables(),

		ConfigOptions: configOptions,

		// We only need that for saving
		kubeLoader:     kubeLoader,
		client:         client,
		generatedSaver: generated.NewConfigLoaderFromDevSpacePath("", baseConfig.Path()),
		log:            log,
	}
}

// Resolve implements interface
func (r *resolver) Resolve(update bool) ([]types.Dependency, error) {
	currentWorkingDirectory, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "get current working directory")
	}

	err = r.resolveRecursive(currentWorkingDirectory, r.DependencyGraph.Root.ID, nil, r.BaseConfig.Dependencies, update)
	if err != nil {
		if _, ok := err.(*cyclicError); ok {
			return nil, err
		}

		return nil, err
	}

	// Save generated
	err = r.generatedSaver.Save(r.BaseCache)
	if err != nil {
		return nil, err
	}

	// get direct childs
	childs := []types.Dependency{}
	for _, v := range r.DependencyGraph.Root.childs {
		childs = append(childs, v.Data.(*Dependency))
	}

	return childs, nil
}

func (r *resolver) resolveRecursive(basePath, parentID string, currentDependency *Dependency, dependencies []*latest.DependencyConfig, update bool) error {
	if currentDependency != nil {
		currentDependency.children = []types.Dependency{}
	}
	for _, dependencyConfig := range dependencies {
		if dependencyConfig.Disabled {
			r.log.Debugf("Skip dependency %s, because it is disabled", dependencyConfig.Name)
			continue
		}

		ID, err := util.GetDependencyID(basePath, dependencyConfig)
		if err != nil {
			return err
		}

		// Try to insert new edge
		var (
			child *Dependency
		)
		if n, ok := r.DependencyGraph.Nodes[ID]; ok {
			err := r.DependencyGraph.addEdge(parentID, ID)
			if err != nil {
				if _, ok := err.(*cyclicError); ok {
					r.log.Warn(err.Error())
				} else {
					return err
				}
			} else {
				child = n.Data.(*Dependency)
			}
		} else {
			child, err = r.resolveDependency(basePath, dependencyConfig, update)
			if err != nil {
				return err
			}

			// is root dependency?
			if currentDependency == nil {
				child.root = true
			}

			_, err = r.DependencyGraph.insertNodeAt(parentID, ID, child)
			if err != nil {
				return errors.Wrap(err, "insert node")
			}

			// load dependencies from dependency
			if !dependencyConfig.IgnoreDependencies && child.localConfig.Config().Dependencies != nil && len(child.localConfig.Config().Dependencies) > 0 {
				err = r.resolveRecursive(child.localPath, ID, child, child.localConfig.Config().Dependencies, update)
				if err != nil {
					return err
				}
			}

			// after we traversed the dependencies initialize the managers with the correct dependencies
			child.registryClient = pullsecrets.NewClient(child.localConfig, child.children, child.kubeClient, child.dockerClient, r.log)
			child.buildController = build.NewController(child.localConfig, child.children, child.kubeClient)
			child.deployController = deploy.NewController(child.localConfig, child.children, child.kubeClient)
		}

		// add child
		if currentDependency != nil {
			currentDependency.children = append(currentDependency.children, child)
		}
	}

	return nil
}

func (r *resolver) resolveDependency(basePath string, dependency *latest.DependencyConfig, update bool) (*Dependency, error) {
	ID, err := util.GetDependencyID(basePath, dependency)
	if err != nil {
		return nil, err
	}

	localPath, err := util.DownloadDependency(ID, basePath, dependency.Source, update, r.log)
	if err != nil {
		return nil, err
	}

	// clone config options
	cloned, err := r.ConfigOptions.Clone()
	if err != nil {
		return nil, errors.Wrap(err, "clone config options")
	}

	// set dependency profile
	cloned.Profiles = []string{}
	cloned.Profiles = append(cloned.Profiles, dependency.ProfileParents...)
	if dependency.Profile != "" {
		cloned.Profiles = append(cloned.Profiles, dependency.Profile)
	}
	cloned.Profiles = append(cloned.Profiles, dependency.Profiles...)
	cloned.DisableProfileActivation = dependency.DisableProfileActivation || r.ConfigOptions.DisableProfileActivation

	// construct load path
	var configPath string
	if dependency.Source.ConfigName != "" {
		configPath = filepath.Join(localPath, dependency.Source.ConfigName)
	} else if strings.HasSuffix(localPath, ".yaml") || strings.HasSuffix(localPath, ".yml") {
		configPath = localPath
		localPath = filepath.Dir(localPath)
	} else {
		configPath = filepath.Join(localPath, constants.DefaultConfigPath)
	}

	// load config
	cloned.GeneratedConfig = nil
	cloned.BasePath = configPath
	if cloned.Vars == nil {
		cloned.Vars = []string{}
	}

	if dependency.OverwriteVars {
		for k, v := range r.BaseVars {
			cloned.Vars = append(cloned.Vars, strings.TrimSpace(k)+"="+strings.TrimSpace(fmt.Sprintf("%v", v)))
		}
	}
	for _, v := range dependency.Vars {
		cloned.Vars = append(cloned.Vars, strings.TrimSpace(v.Name)+"="+strings.TrimSpace(v.Value))
	}

	// load the dependency config
	var dConfigWrapper config.Config
	err = executeInDirectory(filepath.Dir(configPath), func() error {
		dConfigWrapper, err = loader.NewConfigLoader(configPath).LoadWithParser(loader.NewWithCommandsParser(), cloned, r.log)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("loading config for dependency %s", dependency.Name))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	dConfig := dConfigWrapper.Config()

	// set parsed variables in parent config
	if dependency.OverwriteVars {
		for k, v := range dConfigWrapper.Variables() {
			_, ok := r.BaseVars[k]
			if !ok {
				r.BaseVars[k] = v
			}
		}
	}

	// Override complete dev config
	dConfig.Dev = latest.DevConfig{
		Ports:       dConfig.Dev.Ports,
		Sync:        dConfig.Dev.Sync,
		ReplacePods: dConfig.Dev.ReplacePods,
	}

	// Check if we should skip building
	if dependency.SkipBuild {
		for _, b := range dConfig.Images {
			if b.Build == nil {
				b.Build = &latest.BuildConfig{}
			}

			b.Build.Disabled = true
		}
	}

	// Recreate client if necessary
	client := r.client
	if dependency.Namespace != "" {
		if r.client == nil {
			client, err = kubectl.NewClientFromContext("", dependency.Namespace, false, r.kubeLoader)
		} else {
			client, err = kubectl.NewClientFromContext(client.CurrentContext(), dependency.Namespace, false, r.kubeLoader)
		}
		if err != nil {
			return nil, errors.Wrap(err, "create new client")
		}
	}

	// Create docker client
	dockerClient, err := docker.NewClient(r.log)
	if err != nil {
		return nil, errors.Wrap(err, "create docker client")
	}

	// Create registry client for pull secrets
	return &Dependency{
		id:          ID,
		localPath:   localPath,
		localConfig: dConfigWrapper,

		dependencyConfig: dependency,
		dependencyCache:  r.BaseCache,

		kubeClient:     client,
		dockerClient:   dockerClient,
		generatedSaver: generated.NewConfigLoaderFromDevSpacePath(loader.GetLastProfile(cloned.Profiles), configPath),
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
