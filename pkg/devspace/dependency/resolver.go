package dependency

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/generator"
	"github.com/devspace-cloud/devspace/pkg/util/hash"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

// DependencyFolder is the dependency folder in the home directory of the user
const DependencyFolder = ".devspace/dependencies"

// DependencyFolderPath will be filled during init
var DependencyFolderPath string

func init() {
	// Make sure dependency folder exists locally
	homedir, _ := homedir.Dir()

	DependencyFolderPath = filepath.Join(homedir, filepath.FromSlash(DependencyFolder))
}

// ResolverInterface defines the resolver interface that takes dependency configs and resolves them
type ResolverInterface interface {
	Resolve(dependencies []*latest.DependencyConfig, update bool) ([]*Dependency, error)
}

// Resolver implements the resolver interface
type Resolver struct {
	DependencyGraph *Graph

	BasePath   string
	BaseConfig *latest.Config
	BaseCache  *generated.CacheConfig

	AllowCyclic bool

	log log.Logger
}

// NewResolver creates a new resolver for resolving dependencies
func NewResolver(baseConfig *latest.Config, baseCache *generated.CacheConfig, allowCyclic bool, log log.Logger) (*Resolver, error) {
	var id string

	basePath, err := filepath.Abs(".")
	if err != nil {
		return nil, err
	}
	gitRepo := generator.NewGitRepository(basePath, "")
	remote, err := gitRepo.GetRemote()
	if err == nil {
		id = remote
	} else {
		id = basePath
	}

	return &Resolver{
		DependencyGraph: NewGraph(NewNode(id, nil)),

		BaseConfig: baseConfig,
		BaseCache:  baseCache,

		AllowCyclic: allowCyclic,

		log: log,
	}, nil
}

// Resolve implements interface
func (r *Resolver) Resolve(dependencies []*latest.DependencyConfig, update bool) ([]*Dependency, error) {
	r.log.StartWait("Resolving dependencies")
	defer r.log.StopWait()

	currentWorkingDirectory, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "get current working directory")
	}

	err = r.resolveRecursive(currentWorkingDirectory, r.DependencyGraph.Root.ID, dependencies, update)
	if err != nil {
		if _, ok := err.(*CyclicError); ok {
			return nil, err
		}

		return nil, errors.Wrap(err, "resolve dependencies recursive")
	}

	r.log.StopWait()
	r.log.Donef("Resolved %d dependencies", len(r.DependencyGraph.Nodes)-1)
	return r.buildDependencyQueue()
}

func (r *Resolver) buildDependencyQueue() ([]*Dependency, error) {
	retDependencies := make([]*Dependency, 0, len(r.DependencyGraph.Nodes)-1)

	for len(r.DependencyGraph.Nodes) > 1 {
		next := r.DependencyGraph.GetNextLeaf(r.DependencyGraph.Root)
		if next == r.DependencyGraph.Root {
			break
		}

		retDependencies = append(retDependencies, next.Data.(*Dependency))

		err := r.DependencyGraph.RemoveNode(next.ID)
		if err != nil {
			return nil, err
		}
	}

	return retDependencies, nil
}

func (r *Resolver) resolveRecursive(basePath, parentID string, dependencies []*latest.DependencyConfig, update bool) error {
	for _, dependencyConfig := range dependencies {
		ID := r.getDependencyID(basePath, dependencyConfig)

		// Try to insert new edge
		if _, ok := r.DependencyGraph.Nodes[ID]; ok {
			err := r.DependencyGraph.AddEdge(parentID, ID)
			if _, ok := err.(*CyclicError); ok {
				// Check if cyclic dependencies are allowed
				if !r.AllowCyclic {
					return err
				}
			}
		} else {
			dependency, err := r.resolveDependency(basePath, dependencyConfig, update)
			if err != nil {
				return err
			}

			_, err = r.DependencyGraph.InsertNodeAt(parentID, ID, dependency)
			if err != nil {
				return errors.Wrap(err, "insert node")
			}

			// Load dependencies from dependency
			if dependencyConfig.IgnoreDependencies == nil || *dependencyConfig.IgnoreDependencies == false {
				if dependency.Config.Dependencies != nil && len(*dependency.Config.Dependencies) > 0 {
					err = r.resolveRecursive(dependency.LocalPath, ID, *dependency.Config.Dependencies, update)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (r *Resolver) resolveDependency(basePath string, dependency *latest.DependencyConfig, update bool) (*Dependency, error) {
	var (
		ID        = r.getDependencyID(basePath, dependency)
		localPath string
		err       error

		loadConfig = generated.DefaultConfigName
	)

	// Resolve source
	if dependency.Source.Git != nil {
		gitPath := strings.TrimSpace(*dependency.Source.Git)

		os.MkdirAll(DependencyFolderPath, 0755)
		localPath = filepath.Join(DependencyFolderPath, hash.String(gitPath))

		// Check if dependency exists
		_, err := os.Stat(localPath)
		if err != nil {
			update = true
		}

		// Update chart
		if update {
			gitRepo := generator.NewGitRepository(localPath, gitPath)
			_, err := gitRepo.Update()
			if err != nil {
				return nil, errors.Wrap(err, "pull repo")
			}

			r.log.Donef("Pulled %s", ID)
		}
	} else if dependency.Source.Path != nil {
		localPath, err = filepath.Abs(filepath.Join(basePath, filepath.FromSlash(*dependency.Source.Path)))
		if err != nil {
			return nil, errors.Wrap(err, "filepath absolute")
		}
	}

	if dependency.Config != nil {
		loadConfig = *dependency.Config
	}

	// Load config
	dConfig, err := configutil.GetConfigFromPath(localPath, loadConfig, r.BaseCache)
	if err != nil {
		return nil, fmt.Errorf("Error loading config for dependency %s: %v", ID, err)
	}

	// Exchange cluster config
	dConfig.Cluster = r.BaseConfig.Cluster
	dConfig.Dev = &latest.DevConfig{}

	// Load dependency generated config
	dGeneratedConfig, err := generated.LoadConfigFromPath(filepath.Join(localPath, filepath.FromSlash(generated.ConfigPath)))
	if err != nil {
		return nil, fmt.Errorf("Error loading generated config for dependency %s: %v", ID, err)
	}
	dGeneratedConfig.ActiveConfig = loadConfig

	return &Dependency{
		ID:        ID,
		LocalPath: localPath,

		Config:          dConfig,
		GeneratedConfig: dGeneratedConfig,

		DependencyConfig: dependency,
		DependencyCache:  r.BaseCache,
	}, nil
}

func (r *Resolver) getDependencyID(basePath string, dependency *latest.DependencyConfig) string {
	if dependency.Source.Git != nil {
		return strings.TrimSpace(*dependency.Source.Git)
	} else if dependency.Source.Path != nil {
		// Check if it's an git repo
		filePath := filepath.Join(basePath, *dependency.Source.Path)

		gitRepo := generator.NewGitRepository(filePath, "")
		remote, err := gitRepo.GetRemote()
		if err == nil {
			return remote
		}

		return filePath
	}

	return ""
}
