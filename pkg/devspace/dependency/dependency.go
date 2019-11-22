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
	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
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
	PurgeAll(verbose bool) error
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
	UpdateDependencies, SkipPush, ForceDeployDependencies, ForceBuild, Verbose bool
}

// BuildAll will build all dependencies if there are any
func (m *manager) BuildAll(options BuildOptions) error {
	if m.config == nil || m.config.Dependencies == nil || len(m.config.Dependencies) == 0 {
		return nil
	}

	// Resolve all dependencies
	dependencies, err := m.resolver.Resolve(options.UpdateDependencies)
	if err != nil {
		if _, ok := err.(*cyclicError); ok {
			return errors.Errorf("%v.\n To allow cyclic dependencies run with the '%s' flag", err, ansi.Color("--allow-cyclic", "white+b"))
		}

		return err
	}

	defer m.log.StopWait()

	if options.Verbose == false {
		m.log.Infof("To display the complete dependency build run with the '--verbose-dependencies' flag")
	}

	// Deploy all dependencies
	for i := 0; i < len(dependencies); i++ {
		var (
			dependency       = dependencies[i]
			buff             = &bytes.Buffer{}
			dependencyLogger = m.log
		)

		// If not verbose log to a stream
		if options.Verbose == false {
			m.log.StartWait(fmt.Sprintf("Building dependency %d of %d: %s", i+1, len(dependencies), dependency.ID))
			dependencyLogger = log.NewStreamLogger(buff, logrus.InfoLevel)
		} else {
			m.log.Infof(fmt.Sprintf("Building dependency %d of %d: %s", i+1, len(dependencies), dependency.ID))
		}

		err := dependency.Build(options.SkipPush, options.ForceDeployDependencies, options.ForceBuild, dependencyLogger)
		if err != nil {
			return errors.Errorf("Error building dependency %s: %s %v", dependency.ID, buff.String(), err)
		}

		m.log.Donef("Built dependency %s", dependency.ID)
	}

	m.log.StopWait()
	m.log.Donef("Successfully built %d dependencies", len(dependencies))

	return nil
}

// DeployOptions has all options for deploying all dependencies
type DeployOptions struct {
	UpdateDependencies, SkipPush, ForceDeployDependencies, SkipBuild, ForceBuild, ForceDeploy, Verbose bool
}

// DeployAll will deploy all dependencies if there are any
func (m *manager) DeployAll(options DeployOptions) error {
	if m.config == nil || m.config.Dependencies == nil || len(m.config.Dependencies) == 0 {
		return nil
	}

	// Resolve all dependencies
	dependencies, err := m.resolver.Resolve(options.UpdateDependencies)
	if err != nil {
		if _, ok := err.(*cyclicError); ok {
			return errors.Errorf("%v.\n To allow cyclic dependencies run with the '%s' flag", err, ansi.Color("--allow-cyclic", "white+b"))
		}

		return err
	}

	defer m.log.StopWait()

	if options.Verbose == false {
		m.log.Infof("To display the complete dependency deployment run with the '--verbose-dependencies' flag")
	}

	// Deploy all dependencies
	for i := 0; i < len(dependencies); i++ {
		var (
			dependency       = dependencies[i]
			buff             = &bytes.Buffer{}
			dependencyLogger = m.log
		)

		// If not verbose log to a stream
		if options.Verbose == false {
			m.log.StartWait(fmt.Sprintf("Deploying dependency %d of %d: %s", i+1, len(dependencies), dependency.ID))
			dependencyLogger = log.NewStreamLogger(buff, logrus.InfoLevel)
		} else {
			m.log.Infof(fmt.Sprintf("Deploying dependency %d of %d: %s", i+1, len(dependencies), dependency.ID))
		}

		err := dependency.Deploy(options.SkipPush, options.ForceDeployDependencies, options.SkipBuild, options.ForceBuild, options.ForceDeploy, dependencyLogger)
		if err != nil {
			return errors.Errorf("Error deploying dependency %s: %s %v", dependency.ID, buff.String(), err)
		}

		// Prettify path if its a path deployment
		m.log.Donef("Deployed dependency %s", dependency.ID)
	}

	m.log.StopWait()
	m.log.Donef("Successfully deployed %d dependencies", len(dependencies))

	return nil
}

// PurgeAll purges all dependencies in reverse order
func (m *manager) PurgeAll(verbose bool) error {
	if m.config == nil || m.config.Dependencies == nil || len(m.config.Dependencies) == 0 {
		return nil
	}

	// Resolve all dependencies
	dependencies, err := m.resolver.Resolve(false)
	if err != nil {
		if _, ok := err.(*cyclicError); ok {
			return errors.Errorf("%v.\n To allow cyclic dependencies run with the '%s' flag", err, ansi.Color("--allow-cyclic", "white+b"))
		}

		return errors.Wrap(err, "resolve dependencies")
	}

	defer m.log.StopWait()

	if verbose == false {
		m.log.Infof("To display the complete dependency deletion run with the '--verbose-dependencies' flag")
	}

	// Purge all dependencies
	for i := len(dependencies) - 1; i >= 0; i-- {
		var (
			dependency       = dependencies[i]
			buff             = &bytes.Buffer{}
			dependencyLogger = m.log
		)

		// If not verbose log to a stream
		if verbose == false {
			m.log.StartWait(fmt.Sprintf("Purging %d dependencies", i+1))
			dependencyLogger = log.NewStreamLogger(buff, logrus.InfoLevel)
		}

		err := dependency.Purge(dependencyLogger)
		if err != nil {
			return errors.Errorf("Error deploying dependency %s: %s %v", dependency.ID, buff.String(), err)
		}

		m.log.Donef("Purged dependency %s", dependency.ID)
	}

	m.log.StopWait()
	m.log.Donef("Successfully purged %d dependencies", len(dependencies))

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
	buildController  build.Controller
	deployController deploy.Controller
	generatedSaver   generated.ConfigLoader
}

// Build builds and pushes all defined images
func (d *Dependency) Build(skipPush, forceDependencies, forceBuild bool, log log.Logger) error {
	// Check if we should redeploy
	directoryHash, err := hash.DirectoryExcludes(d.LocalPath, []string{".git", ".devspace"}, true)
	if err != nil {
		return errors.Wrap(err, "hash directory")
	}

	// Check if we skip the dependency deploy
	if forceDependencies == false && directoryHash == d.DependencyCache.GetActive().Dependencies[d.ID] {
		return nil
	}

	d.DependencyCache.GetActive().Dependencies[d.ID] = directoryHash

	// Switch current working directory
	currentWorkingDirectory, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "getwd")
	}

	err = os.Chdir(d.LocalPath)
	if err != nil {
		return errors.Wrap(err, "change working directory")
	}

	// Change back to original working directory
	defer os.Chdir(currentWorkingDirectory)

	// Check if image build is enabled
	builtImages := make(map[string]string)
	if d.DependencyConfig.SkipBuild == nil || *d.DependencyConfig.SkipBuild == false {
		// Build images
		builtImages, err = d.buildController.Build(&build.Options{
			SkipPush:     skipPush,
			ForceRebuild: forceBuild,
		}, log)
		if err != nil {
			return err
		}

		// Save config if an image was built
		if len(builtImages) > 0 {
			err := d.generatedSaver.Save(d.GeneratedConfig)
			if err != nil {
				return errors.Errorf("Error saving generated config: %v", err)
			}
		}
	}

	log.Donef("Built dependency %s", d.ID)
	return nil
}

// Deploy deploys the dependency if necessary
func (d *Dependency) Deploy(skipPush, forceDependencies, skipBuild, forceBuild, forceDeploy bool, log log.Logger) error {
	// Check if we should redeploy
	directoryHash, err := hash.DirectoryExcludes(d.LocalPath, []string{".git", ".devspace"}, true)
	if err != nil {
		return errors.Wrap(err, "hash directory")
	}

	// Check if we skip the dependency deploy
	if forceDependencies == false && directoryHash == d.DependencyCache.GetActive().Dependencies[d.ID] {
		return nil
	}

	d.DependencyCache.GetActive().Dependencies[d.ID] = directoryHash

	// Switch current working directory
	currentWorkingDirectory, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "getwd")
	}

	err = os.Chdir(d.LocalPath)
	if err != nil {
		return errors.Wrap(err, "change working directory")
	}

	// Change back to original working directory
	defer os.Chdir(currentWorkingDirectory)

	// Create namespace if necessary
	err = d.kubeClient.EnsureDefaultNamespace(log)
	if err != nil {
		return errors.Errorf("Unable to create namespace: %v", err)
	}

	// Create docker client
	dockerClient, err := docker.NewClient(log)
	if err != nil {
		return errors.Wrap(err, "create docker client")
	}

	// Create pull secrets and private registry if necessary
	registryClient := registry.NewClient(d.Config, d.kubeClient, dockerClient, log)
	err = registryClient.CreatePullSecrets()
	if err != nil {
		return err
	}

	// Check if image build is enabled
	builtImages := make(map[string]string)
	if skipBuild == false && (d.DependencyConfig.SkipBuild == nil || *d.DependencyConfig.SkipBuild == false) {
		// Build images
		builtImages, err = d.buildController.Build(&build.Options{
			SkipPush:     skipPush,
			ForceRebuild: forceBuild,
		}, log)
		if err != nil {
			return err
		}

		// Save config if an image was built
		if len(builtImages) > 0 {
			err := d.generatedSaver.Save(d.GeneratedConfig)
			if err != nil {
				return errors.Errorf("Error saving generated config: %v", err)
			}
		}
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

// Purge purges the dependency
func (d *Dependency) Purge(log log.Logger) error {
	// Switch current working directory
	currentWorkingDirectory, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "getwd")
	}

	err = os.Chdir(d.LocalPath)
	if err != nil {
		return errors.Wrap(err, "change working directory")
	}

	defer func() {
		// Change back to original working directory
		os.Chdir(currentWorkingDirectory)
	}()

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
