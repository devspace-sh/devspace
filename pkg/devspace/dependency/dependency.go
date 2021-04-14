package dependency

import (
	"bytes"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/command"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/util/exit"
	"io"
	"mvdan.cc/sh/v3/interp"
	"os"

	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/pullsecrets"
	"github.com/loft-sh/devspace/pkg/util/hash"
	"github.com/loft-sh/devspace/pkg/util/log"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

	// Command executes a dependency command
	Command(options CommandOptions) error
}

type manager struct {
	config       *latest.Config
	cache        *generated.CacheConfig
	log          log.Logger
	resolver     ResolverInterface
	hookExecuter hook.Executer
	client       kubectl.Client
}

// NewManager creates a new instance of the interface Manager
func NewManager(config config.Config, client kubectl.Client, allowCyclic bool, configOptions *loader.ConfigOptions, logger log.Logger) Manager {
	return &manager{
		config:       config.Config(),
		cache:        config.Generated().GetActive(),
		log:          logger,
		resolver:     NewResolver(config.Config(), config.Generated(), client, allowCyclic, configOptions, logger),
		hookExecuter: hook.NewExecuter(config, nil),
		client:       client,
	}
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

type ResolveOptions struct {
	Dependencies       []string
	UpdateDependencies bool
	Verbose            bool
}

func (m *manager) ResolveAll(options ResolveOptions) ([]types.Dependency, error) {
	dependencies, err := m.handleDependencies(options.Dependencies, false, options.UpdateDependencies, false, options.Verbose, "Resolve", func(dependency *Dependency, log log.Logger) error {
		return nil
	})
	if err != nil {
		return nil, err
	}

	return dependencies, nil
}

// CommandOptions has all options for executing a command from a dependency
type CommandOptions struct {
	Dependencies       []string
	Command            string
	Args               []string
	UpdateDependencies bool
	Verbose            bool
}

// Command will execute a dependency command
func (m *manager) Command(options CommandOptions) error {
	_, err := m.handleDependencies(options.Dependencies, false, options.UpdateDependencies, true, options.Verbose, "Command", func(dependency *Dependency, log log.Logger) error {
		// Switch current working directory
		currentWorkingDirectory, err := dependency.prepare(true)
		if err != nil {
			return err
		} else if currentWorkingDirectory == "" {
			return nil
		}

		// Change back to original working directory
		defer os.Chdir(currentWorkingDirectory)

		return ExecuteCommand(dependency.localConfig.Config().Commands, options.Command, options.Args)
	})
	return err
}

// ExecuteCommand executes a given command from the available commands
func ExecuteCommand(commands []*latest.CommandConfig, cmd string, args []string) error {
	err := command.ExecuteCommand(commands, cmd, args)
	if err != nil {
		shellExitError, ok := err.(interp.ShellExitStatus)
		if ok {
			return &exit.ReturnCodeError{
				ExitCode: int(shellExitError),
			}
		}

		exitError, ok := err.(interp.ExitStatus)
		if ok {
			return &exit.ReturnCodeError{
				ExitCode: int(exitError),
			}
		}

		return errors.Wrap(err, "execute command")
	}

	return nil
}

// BuildOptions has all options for building all dependencies
type BuildOptions struct {
	BuildOptions build.Options

	Dependencies            []string
	UpdateDependencies      bool
	ForceDeployDependencies bool
	Verbose                 bool
}

// BuildAll will build all dependencies if there are any
func (m *manager) BuildAll(options BuildOptions) ([]types.Dependency, error) {
	return m.handleDependencies(options.Dependencies, false, options.UpdateDependencies, false, options.Verbose, "Build", func(dependency *Dependency, log log.Logger) error {
		return dependency.Build(options.ForceDeployDependencies, &options.BuildOptions, log)
	})
}

// DeployOptions has all options for deploying all dependencies
type DeployOptions struct {
	BuildOptions build.Options

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
	err := m.hookExecuter.Execute(hook.Before, hook.StageDependencies, hook.All, hook.Context{Client: m.client}, m.log)
	if err != nil {
		return nil, err
	}

	dependencies, err := m.handleDependencies(options.Dependencies, false, options.UpdateDependencies, false, options.Verbose, "Deploy", func(dependency *Dependency, log log.Logger) error {
		return dependency.Deploy(options.ForceDeployDependencies, options.SkipBuild, options.SkipDeploy, options.ForceDeploy, &options.BuildOptions, log)
	})
	if err != nil {
		m.hookExecuter.OnError(hook.StageDependencies, []string{hook.All}, hook.Context{Client: m.client, Error: err}, m.log)
		return nil, err
	}

	err = m.hookExecuter.Execute(hook.After, hook.StageDependencies, hook.All, hook.Context{Client: m.client}, m.log)
	if err != nil {
		return nil, err
	}

	return dependencies, nil
}

// PurgeOptions has all options for purging all dependencies
type PurgeOptions struct {
	Dependencies []string
	Verbose      bool
}

// PurgeAll purges all dependencies in reverse order
func (m *manager) PurgeAll(options PurgeOptions) ([]types.Dependency, error) {
	return m.handleDependencies(options.Dependencies, true, false, false, options.Verbose, "Purge", func(dependency *Dependency, log log.Logger) error {
		return dependency.Purge(log)
	})
}

// RenderOptions has all options for rendering all dependencies
type RenderOptions struct {
	Dependencies       []string
	Verbose            bool
	UpdateDependencies bool
	SkipBuild          bool
	Writer             io.Writer

	BuildOptions build.Options
}

func (m *manager) RenderAll(options RenderOptions) ([]types.Dependency, error) {
	return m.handleDependencies(options.Dependencies, false, options.UpdateDependencies, false, options.Verbose, "Render", func(dependency *Dependency, log log.Logger) error {
		return dependency.Render(options.SkipBuild, &options.BuildOptions, options.Writer, log)
	})
}

func (m *manager) handleDependencies(filterDependencies []string, reverse, updateDependencies, silent, verbose bool, actionName string, action func(dependency *Dependency, log log.Logger) error) ([]types.Dependency, error) {
	if m.config == nil || m.config.Dependencies == nil || len(m.config.Dependencies) == 0 {
		return nil, nil
	}

	if silent == false {
		m.log.Infof("Start resolving dependencies")
	}

	// Resolve all dependencies
	dependencies, err := m.resolver.Resolve(updateDependencies)
	if err != nil {
		if _, ok := err.(*cyclicError); ok {
			return nil, errors.Errorf("%v.\n To allow cyclic dependencies run with the '%s' flag", err, ansi.Color("--allow-cyclic", "white+b"))
		}

		return nil, errors.Wrap(err, "resolve dependencies")
	}

	defer m.log.StopWait()

	if silent == false {
		m.log.Donef("Resolved %d dependencies", len(dependencies))
	}
	if silent == false && verbose == false {
		m.log.Infof("To display the complete dependency execution log run with the '--verbose-dependencies' flag")
	}

	// Execute all dependencies
	i := 0
	if reverse {
		i = len(dependencies) - 1
	}

	numDependencies := len(dependencies)
	if len(filterDependencies) > 0 {
		numDependencies = len(filterDependencies)
	}

	executedDependencies := []types.Dependency{}
	if silent == false {
		m.log.StartWait(fmt.Sprintf("%s %d dependencies", actionName, numDependencies))
	}
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
		if foundDependency(dependency.dependencyConfig.Name, filterDependencies) == false {
			continue
		}

		// If not verbose log to a stream
		if verbose == false {
			dependencyLogger = log.NewStreamLogger(buff, logrus.InfoLevel)
		}

		err := action(dependency, dependencyLogger)
		if err != nil {
			return nil, errors.Wrapf(err, "%s dependency %s error %s", actionName, dependency.id, buff.String())
		}

		executedDependencies = append(executedDependencies, dependency)
		if silent == false {
			m.log.Donef("%s dependency %s completed", actionName, dependency.id)
		}
	}
	m.log.StopWait()
	if silent == false {
		if len(executedDependencies) > 0 {
			m.log.Donef("Successfully processed %d dependencies", len(executedDependencies))
		} else {
			m.log.Done("No dependency processed")
		}
	}

	return executedDependencies, nil
}

// Dependency holds the dependency config and has an id
type Dependency struct {
	id          string
	localPath   string
	localConfig config.Config

	dependencyConfig *latest.DependencyConfig
	dependencyCache  *generated.Config

	kubeClient       kubectl.Client
	registryClient   pullsecrets.Client
	buildController  build.Controller
	deployController deploy.Controller
	generatedSaver   generated.ConfigLoader
}

// Implement Interface Methods

func (d *Dependency) ID() string { return d.id }

func (d *Dependency) NameOrID() string {
	if d.dependencyConfig.Name != "" {
		return d.dependencyConfig.Name
	}
	return d.id
}

func (d *Dependency) Config() config.Config { return d.localConfig }

func (d *Dependency) LocalPath() string { return d.localPath }

func (d *Dependency) DependencyConfig() *latest.DependencyConfig { return d.dependencyConfig }

// Build builds and pushes all defined images
func (d *Dependency) Build(forceDependencies bool, buildOptions *build.Options, log log.Logger) error {
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
	_, err = d.buildImages(false, buildOptions, log)
	if err != nil {
		return err
	}

	log.Donef("Built dependency %s", d.id)
	return nil
}

// Deploy deploys the dependency if necessary
func (d *Dependency) Deploy(forceDependencies, skipBuild, skipDeploy, forceDeploy bool, buildOptions *build.Options, log log.Logger) error {
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
	err = d.kubeClient.EnsureDeployNamespaces(d.localConfig.Config(), log)
	if err != nil {
		return errors.Errorf("Unable to create namespace: %v", err)
	}

	// Create pull secrets and private registry if necessary
	err = d.registryClient.CreatePullSecrets()
	if err != nil {
		log.Warn(err)
	}

	// Check if image build is enabled
	builtImages, err := d.buildImages(skipBuild, buildOptions, log)
	if err != nil {
		return err
	}

	// Deploy all defined deployments
	if skipDeploy == false {
		err = d.deployController.Deploy(&deploy.Options{
			ForceDeploy: forceDeploy,
			BuiltImages: builtImages,
		}, log)
		if err != nil {
			return err
		}
	}

	// Save Config
	err = d.generatedSaver.Save(d.localConfig.Generated())
	if err != nil {
		return errors.Errorf("Error saving generated config: %v", err)
	}

	log.Donef("Deployed dependency %s", d.id)
	return nil
}

// Render renders the dependency
func (d *Dependency) Render(skipBuild bool, buildOptions *build.Options, out io.Writer, log log.Logger) error {
	// Switch current working directory
	currentWorkingDirectory, err := d.changeWorkingDirectory()
	if err != nil {
		return errors.Wrap(err, "getwd")
	}

	defer os.Chdir(currentWorkingDirectory)

	// Check if image build is enabled
	builtImages, err := d.buildImages(skipBuild, buildOptions, log)
	if err != nil {
		return err
	}

	// Deploy all defined deployments
	return d.deployController.Render(&deploy.Options{
		BuiltImages: builtImages,
	}, out, log)
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
		log.Errorf("Error purging dependency %s: %v", d.id, err)
	}

	err = d.generatedSaver.Save(d.localConfig.Generated())
	if err != nil {
		log.Errorf("Error saving generated.yaml: %v", err)
	}

	delete(d.dependencyCache.GetActive().Dependencies, d.id)
	log.Donef("Purged dependency %s", d.id)
	return nil
}

func (d *Dependency) buildImages(skipBuild bool, buildOptions *build.Options, log log.Logger) (map[string]string, error) {
	var err error

	// Check if image build is enabled
	builtImages := make(map[string]string)
	if skipBuild == false && d.dependencyConfig.SkipBuild == false {
		// Build images
		builtImages, err = d.buildController.Build(buildOptions, log)
		if err != nil {
			return nil, err
		}

		// Save config if an image was built
		if len(builtImages) > 0 {
			err := d.generatedSaver.Save(d.localConfig.Generated())
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

	err = os.Chdir(d.localPath)
	if err != nil {
		return "", errors.Wrap(err, "change working directory")
	}

	return currentWorkingDirectory, nil
}

func (d *Dependency) prepare(forceDependencies bool) (string, error) {
	// Check if we should redeploy
	directoryHash, err := hash.DirectoryExcludes(d.localPath, []string{".git", ".devspace"}, true)
	if err != nil {
		return "", errors.Wrap(err, "hash directory")
	}

	// Check if we skip the dependency deploy
	if forceDependencies == false && directoryHash == d.dependencyCache.GetActive().Dependencies[d.id] {
		return "", nil
	}

	d.dependencyCache.GetActive().Dependencies[d.id] = directoryHash
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
