package dependency

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/build"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"

	"github.com/devspace-cloud/devspace/pkg/util/git"
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
	Resolve(update bool) ([]*Dependency, error)
}

// Resolver implements the resolver interface
type resolver struct {
	DependencyGraph *graph

	BasePath   string
	BaseConfig *latest.Config
	BaseCache  *generated.Config

	ConfigOptions *loader.ConfigOptions
	AllowCyclic   bool

	client         kubectl.Client
	generatedSaver generated.ConfigLoader
	log            log.Logger
}

// NewResolver creates a new resolver for resolving dependencies
func NewResolver(baseConfig *latest.Config, baseCache *generated.Config, client kubectl.Client, allowCyclic bool, configOptions *loader.ConfigOptions, log log.Logger) (ResolverInterface, error) {
	var id string

	basePath, err := filepath.Abs(".")
	if err != nil {
		return nil, err
	}
	gitRepo := git.NewGitRepository(basePath, "")
	remote, err := gitRepo.GetRemote()
	if err == nil {
		id = remote
	} else {
		id = basePath
	}

	return &resolver{
		DependencyGraph: newGraph(newNode(id, nil)),

		BaseConfig: baseConfig,
		BaseCache:  baseCache,

		AllowCyclic:   allowCyclic,
		ConfigOptions: configOptions,

		// We only need that for saving
		generatedSaver: generated.NewConfigLoader(""),
		client:         client,
		log:            log,
	}, nil
}

// Resolve implements interface
func (r *resolver) Resolve(update bool) ([]*Dependency, error) {
	r.log.Info("Start resolving dependencies")

	currentWorkingDirectory, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "get current working directory")
	}

	err = r.resolveRecursive(currentWorkingDirectory, r.DependencyGraph.Root.ID, r.BaseConfig.Dependencies, update)
	if err != nil {
		if _, ok := err.(*cyclicError); ok {
			return nil, err
		}

		return nil, errors.Wrap(err, "resolve dependencies recursive")
	}

	r.log.Donef("Resolved %d dependencies", len(r.DependencyGraph.Nodes)-1)

	// Save generated
	err = r.generatedSaver.Save(r.BaseCache)
	if err != nil {
		return nil, err
	}

	return r.buildDependencyQueue()
}

func (r *resolver) buildDependencyQueue() ([]*Dependency, error) {
	retDependencies := make([]*Dependency, 0, len(r.DependencyGraph.Nodes)-1)

	for len(r.DependencyGraph.Nodes) > 1 {
		next := r.DependencyGraph.getNextLeaf(r.DependencyGraph.Root)
		if next == r.DependencyGraph.Root {
			break
		}

		retDependencies = append(retDependencies, next.Data.(*Dependency))

		err := r.DependencyGraph.removeNode(next.ID)
		if err != nil {
			return nil, err
		}
	}

	return retDependencies, nil
}

func (r *resolver) resolveRecursive(basePath, parentID string, dependencies []*latest.DependencyConfig, update bool) error {
	for _, dependencyConfig := range dependencies {
		ID := r.getDependencyID(basePath, dependencyConfig)

		// Try to insert new edge
		if _, ok := r.DependencyGraph.Nodes[ID]; ok {
			err := r.DependencyGraph.addEdge(parentID, ID)
			if err != nil {
				if _, ok := err.(*cyclicError); ok {
					// Check if cyclic dependencies are allowed
					if !r.AllowCyclic {
						return err
					}
				} else {
					return err
				}
			}
		} else {
			dependency, err := r.resolveDependency(basePath, dependencyConfig, update)
			if err != nil {
				return err
			}

			_, err = r.DependencyGraph.insertNodeAt(parentID, ID, dependency)
			if err != nil {
				return errors.Wrap(err, "insert node")
			}

			// Load dependencies from dependency
			if dependencyConfig.IgnoreDependencies == nil || *dependencyConfig.IgnoreDependencies == false {
				if dependency.Config.Dependencies != nil && len(dependency.Config.Dependencies) > 0 {
					err = r.resolveRecursive(dependency.LocalPath, ID, dependency.Config.Dependencies, update)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (r *resolver) resolveDependency(basePath string, dependency *latest.DependencyConfig, update bool) (*Dependency, error) {
	var (
		ID        = r.getDependencyID(basePath, dependency)
		localPath string
		err       error
	)

	// Resolve source
	if dependency.Source.Git != "" {
		gitPath := strings.TrimSpace(dependency.Source.Git)

		os.MkdirAll(DependencyFolderPath, 0755)
		localPath = filepath.Join(DependencyFolderPath, hash.String(ID))

		// Check if dependency exists
		_, err := os.Stat(localPath)
		if err != nil {
			update = true
		}

		// Update dependency
		if update {
			var (
				gitRepo  = git.NewGitRepository(localPath, gitPath)
				tag      = dependency.Source.Tag
				branch   = dependency.Source.Branch
				revision = dependency.Source.Revision
			)

			err = gitRepo.Update(tag == "" && branch == "" && revision == "")
			if err != nil {
				return nil, errors.Wrap(err, "pull repo")
			}

			if tag != "" || branch != "" || revision != "" {
				err = gitRepo.Checkout(tag, branch, revision)
				if err != nil {
					return nil, errors.Wrap(err, "checkout")
				}
			}

			r.log.Donef("Pulled %s", ID)
		}
	} else if dependency.Source.Path != "" {
		localPath, err = filepath.Abs(filepath.Join(basePath, filepath.FromSlash(dependency.Source.Path)))
		if err != nil {
			return nil, errors.Wrap(err, "filepath absolute")
		}
	}

	if dependency.Source.SubPath != "" {
		localPath = filepath.Join(localPath, filepath.FromSlash(dependency.Source.SubPath))
	}

	// Clone config options
	cloned, err := r.ConfigOptions.Clone()
	if err != nil {
		return nil, errors.Wrap(err, "clone config options")
	}

	cloned.Profile = dependency.Profile

	// Construct load path
	configPath := filepath.Join(localPath, constants.DefaultConfigPath)

	// Load config
	configLoader := loader.NewConfigLoader(cloned, log.Discard)
	dConfig, err := configLoader.LoadFromPath(r.BaseCache, configPath)
	if err != nil {
		return nil, errors.Errorf("Error loading config for dependency %s: %v", ID, err)
	}

	// Override complete dev config
	dConfig.Dev = &latest.DevConfig{}

	// Check if we should skip building
	if dependency.SkipBuild != nil && *dependency.SkipBuild == true {
		dConfig.Images = map[string]*latest.ImageConfig{}
	}

	// Load dependency generated config
	gLoader := generated.NewConfigLoader(dependency.Profile)
	dGeneratedConfig, err := gLoader.LoadFromPath(filepath.Join(localPath, filepath.FromSlash(generated.ConfigPath)))
	if err != nil {
		return nil, errors.Errorf("Error loading generated config for dependency %s: %v", ID, err)
	}

	dGeneratedConfig.ActiveProfile = dependency.Profile
	generated.InitDevSpaceConfig(dGeneratedConfig, dependency.Profile)

	// Recreate client if necessary
	client := r.client
	if dependency.Namespace != "" {
		client, err = kubectl.NewClientFromContext(client.CurrentContext(), dependency.Namespace, false)
		if err != nil {
			return nil, errors.Wrap(err, "create new client")
		}
	}

	return &Dependency{
		ID:        ID,
		LocalPath: localPath,

		Config:          dConfig,
		GeneratedConfig: dGeneratedConfig,

		DependencyConfig: dependency,
		DependencyCache:  r.BaseCache,

		kubeClient: client,

		buildController:  build.NewController(dConfig, dGeneratedConfig.GetActive(), client),
		deployController: deploy.NewController(dConfig, dGeneratedConfig.GetActive(), client),
		generatedSaver:   gLoader,
	}, nil
}

var authRegEx = regexp.MustCompile("^(https?:\\/\\/)[^:]+:[^@]+@(.*)$")

func (r *resolver) getDependencyID(basePath string, dependency *latest.DependencyConfig) string {
	if dependency.Source.Git != "" {
		// Erase authentication credentials
		id := strings.TrimSpace(dependency.Source.Git)
		id = authRegEx.ReplaceAllString(id, "$1$2")

		if dependency.Source.Tag != "" {
			id += "@" + dependency.Source.Tag
		} else if dependency.Source.Branch != "" {
			id += "@" + dependency.Source.Branch
		} else if dependency.Source.Revision != "" {
			id += "@" + dependency.Source.Revision
		}

		if dependency.Source.SubPath != "" {
			id += ":" + dependency.Source.SubPath
		}

		if dependency.Profile != "" {
			id += " - profile " + dependency.Profile
		}

		return id
	} else if dependency.Source.Path != "" {
		// Check if it's an git repo
		filePath := filepath.Join(basePath, dependency.Source.Path)

		gitRepo := git.NewGitRepository(filePath, "")
		remote, err := gitRepo.GetRemote()
		if err == nil {
			return remote
		}

		if dependency.Profile != "" {
			filePath += " - profile " + dependency.Profile
		}

		return filePath
	}

	return ""
}
