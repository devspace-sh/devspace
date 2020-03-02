package dependency

import (
	"bytes"
	"fmt"
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/build"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/registry"
	"github.com/devspace-cloud/devspace/pkg/util/hash"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Manager can update, build, deploy and purge dependencies.
type Manager interface {
	UpdateAll() error
	BuildAll(options BuildOptions) error
	DeployAll(options DeployOptions) error
	PurgeAll(options PurgeOptions) error
	RenderAll(options RenderOptions) error
}

type manager struct {
	config   *latest.Config
	log      log.Logger
	resolver ResolverInterface
}

// NewManager creates a new instance of the interface Manager
func NewManager(config *latest.Config, cache *generated.Config, client kubectl.Client, allowCyclic bool, configOptions *loader.ConfigOptions, logger log.Logger) (Manager, error) {
	resolver, err := NewResolver(config, cache, client, allowCyclic, configOptions, logger)
	if err != nil {
		return nil, errors.Wrap(err, "new resolver")
	}

	return &manager{
		config:   config,
		log:      logger,
		resolver: resolver,
	}, nil
}

// UpdateAll will update all dependencies if there are any
func (m *manager) UpdateAll() error {
	if m.config == nil || m.config.Dependencies == nil || len(m.config.Dependencies) == 0 {
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

// BuildOptions has all options for building all dependencies
type BuildOptions struct {
	Dependencies            []string
	UpdateDependencies      bool
	SkipPush                bool
	ForceDeployDependencies bool
	ForceBuild              bool
	Verbose                 bool
}

// BuildAll will build all dependencies if there are any
func (m *manager) BuildAll(options BuildOptions) error {
	return m.handleDependencies(options.Dependencies, false, options.UpdateDependencies, options.Verbose, "Build", func(dependency *Dependency, log log.Logger) error {
		return dependency.Build(options.SkipPush, options.ForceDeployDependencies, options.ForceBuild, log)
	})
}

// DeployOptions has all options for deploying all dependencies
type DeployOptions struct {
	Dependencies            []string
	UpdateDependencies      bool
	SkipPush                bool
	ForceDeployDependencies bool
	SkipBuild               bool
	ForceBuild              bool
	ForceDeploy             bool
	Verbose                 bool
}

// DeployAll will deploy all dependencies if there are any
func (m *manager) DeployAll(options DeployOptions) error {
	return m.handleDependencies(options.Dependencies, false, options.UpdateDependencies, options.Verbose, "Deploy", func(dependency *Dependency, log log.Logger) error {
		return dependency.Deploy(options.SkipPush, options.ForceDeployDependencies, options.SkipBuild, options.ForceBuild, options.ForceDeploy, log)
	})
}

// PurgeOptions has all options for purging all dependencies
type PurgeOptions struct {
	Dependencies []string
	Verbose      bool
}

// PurgeAll purges all dependencies in reverse order
func (m *manager) PurgeAll(options PurgeOptions) error {
	return m.handleDependencies(options.Dependencies, true, false, options.Verbose, "Purge", func(dependency *Dependency, log log.Logger) error {
		return dependency.Purge(log)
	})
}

// RenderOptions has all options for rendering all dependencies
type RenderOptions struct {
	Dependencies       []string
	Verbose            bool
	UpdateDependencies bool
	SkipPush           bool
	SkipBuild          bool
	ForceBuild         bool
}

func (m *manager) RenderAll(options RenderOptions) error {
	return m.handleDependencies(options.Dependencies, false, options.UpdateDependencies, options.Verbose, "Render", func(dependency *Dependency, log log.Logger) error {
		return dependency.Render(options.SkipPush, options.SkipBuild, options.ForceBuild, log)
	})
}

func (m *manager) handleDependencies(filterDependencies []string, reverse, updateDependencies, verbose bool, actionName string, action func(dependency *Dependency, log log.Logger) error) error {
	if m.config == nil || m.config.Dependencies == nil || len(m.config.Dependencies) == 0 {
		return nil
	}

	// Resolve all dependencies
	dependencies, err := m.resolver.Resolve(updateDependencies)
	if err != nil {
		if _, ok := err.(*cyclicError); ok {
			return errors.Errorf("%v.\n To allow cyclic dependencies run with the '%s' flag", err, ansi.Color("--allow-cyclic", "white+b"))
		}

		return errors.Wrap(err, "resolve dependencies")
	}

	defer m.log.StopWait()

	if verbose == false {
		m.log.Infof("To display the complete dependency execution log run with the '--verbose-dependencies' flag")
	}

	// Execute all dependencies
	i := 0
	if reverse {
		i = len(dependencies) - 1
	}

	executed := 0
	m.log.StartWait(fmt.Sprintf("%s %d dependencies", actionName, len(dependencies)))
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

		// Check if we should act on this dependency
		if foundDependency(dependency.DependencyConfig.Name, filterDependencies) == false {
			continue
		}

		// If not verbose log to a stream
		if verbose == false {
			dependencyLogger = log.NewStreamLogger(buff, logrus.InfoLevel)
		}

		err := action(dependency, dependencyLogger)
		if err != nil {
			return errors.Errorf("%s dependency %s error: %s %v", actionName, dependency.ID, buff.String(), err)
		}

		executed++
		m.log.Donef("%s dependency %s completed", actionName, dependency.ID)
	}
	m.log.StopWait()

	if executed > 0 {
		m.log.Donef("Successfully processed %d dependencies", executed)
	} else {
		m.log.Done("No dependency processed")
	}

	return nil
}

// Dependency holds the dependency config and has an id
type Dependency struct {
	ID              string
	LocalPath       string
	Config          *latest.Config
	GeneratedConfig *generated.Config

	DependencyConfig *latest.DependencyConfig
	DependencyCache  *generated.Config

	kubeClient       kubectl.Client
	registryClient   registry.Client
	buildController  build.Controller
	deployController deploy.Controller
	generatedSaver   generated.ConfigLoader
}

// Build builds and pushes all defined images
func (d *Dependency) Build(skipPush, forceDependencies, forceBuild bool, log log.Logger) error {
	// Switch current working directory
	currentWorkingDirectory, err := d.prepare(forceDependencies)
	if err != nil {
		return err
	} else if currentWorkingDirectory == "" {
		return nil
	}

	// Change back to original working directory
	defer os.Chdir(currentWorkingDirectory)

	// Check if image build is enabled
	_, err = d.buildImages(false, skipPush, forceBuild, log)
	if err != nil {
		return err
	}

	log.Donef("Built dependency %s", d.ID)
	return nil
}

// Deploy deploys the dependency if necessary
func (d *Dependency) Deploy(skipPush, forceDependencies, skipBuild, forceBuild, forceDeploy bool, log log.Logger) error {
	// Switch current working directory
	currentWorkingDirectory, err := d.prepare(forceDependencies)
	if err != nil {
		return err
	} else if currentWorkingDirectory == "" {
		return nil
	}

	// Change back to original working directory
	defer os.Chdir(currentWorkingDirectory)

	// Create namespace if necessary
	err = d.kubeClient.EnsureDefaultNamespace(log)
	if err != nil {
		return errors.Errorf("Unable to create namespace: %v", err)
	}

	// Create pull secrets and private registry if necessary
	err = d.registryClient.CreatePullSecrets()
	if err != nil {
		log.Warn(err)
	}

	// Check if image build is enabled
	builtImages, err := d.buildImages(skipBuild, skipPush, forceBuild, log)
	if err != nil {
		return err
	}

	// Deploy all defined deployments
	err = d.deployController.Deploy(&deploy.Options{
		ForceDeploy: forceDeploy,
		BuiltImages: builtImages,
	}, log)
	if err != nil {
		return err
	}

	// Save Config
	err = d.generatedSaver.Save(d.GeneratedConfig)
	if err != nil {
		return errors.Errorf("Error saving generated config: %v", err)
	}

	log.Donef("Deployed dependency %s", d.ID)
	return nil
}

// Render renders the dependency
func (d *Dependency) Render(skipPush, skipBuild, forceBuild bool, log log.Logger) error {
	// Switch current working directory
	currentWorkingDirectory, err := d.changeWorkingDirectory()
	if err != nil {
		return errors.Wrap(err, "getwd")
	}

	defer os.Chdir(currentWorkingDirectory)

	// Check if image build is enabled
	builtImages, err := d.buildImages(skipBuild, skipPush, forceBuild, log)
	if err != nil {
		return err
	}

	// Deploy all defined deployments
	return d.deployController.Render(&deploy.Options{
		BuiltImages: builtImages,
	}, os.Stdout)
}

// Purge purges the dependency
func (d *Dependency) Purge(log log.Logger) error {
	// Switch current working directory
	currentWorkingDirectory, err := d.changeWorkingDirectory()
	if err != nil {
		return errors.Wrap(err, "getwd")
	}

	defer os.Chdir(currentWorkingDirectory)

	// Purge the deployments
	err = d.deployController.Purge(nil, log)
	if err != nil {
		log.Errorf("Error purging dependency %s: %v", d.ID, err)
	}

	err = d.generatedSaver.Save(d.GeneratedConfig)
	if err != nil {
		log.Errorf("Error saving generated.yaml: %v", err)
	}

	delete(d.DependencyCache.GetActive().Dependencies, d.ID)
	log.Donef("Purged dependency %s", d.ID)
	return nil
}

func (d *Dependency) buildImages(skipBuild, skipPush, forceBuild bool, log log.Logger) (map[string]string, error) {
	var err error

	// Check if image build is enabled
	builtImages := make(map[string]string)
	if skipBuild == false && (d.DependencyConfig.SkipBuild == nil || *d.DependencyConfig.SkipBuild == false) {
		// Build images
		builtImages, err = d.buildController.Build(&build.Options{
			SkipPush:     skipPush,
			ForceRebuild: forceBuild,
		}, log)
		if err != nil {
			return nil, err
		}

		// Save config if an image was built
		if len(builtImages) > 0 {
			err := d.generatedSaver.Save(d.GeneratedConfig)
			if err != nil {
				return nil, errors.Errorf("Error saving generated config: %v", err)
			}
		}
	}

	return builtImages, nil
}

func (d *Dependency) changeWorkingDirectory() (string, error) {
	// Switch current working directory
	currentWorkingDirectory, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "getwd")
	}

	err = os.Chdir(d.LocalPath)
	if err != nil {
		return "", errors.Wrap(err, "change working directory")
	}

	return currentWorkingDirectory, nil
}

func (d *Dependency) prepare(forceDependencies bool) (string, error) {
	// Check if we should redeploy
	directoryHash, err := hash.DirectoryExcludes(d.LocalPath, []string{".git", ".devspace"}, true)
	if err != nil {
		return "", errors.Wrap(err, "hash directory")
	}

	// Check if we skip the dependency deploy
	if forceDependencies == false && directoryHash == d.DependencyCache.GetActive().Dependencies[d.ID] {
		return "", nil
	}

	d.DependencyCache.GetActive().Dependencies[d.ID] = directoryHash
	return d.changeWorkingDirectory()
}

func foundDependency(name string, dependencies []string) bool {
	if len(dependencies) == 0 {
		return true
	}

	for _, n := range dependencies {
		if n == name {
			return true
		}
	}

	return false
}
