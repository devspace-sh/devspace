package dependency

import (
	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/pullsecrets"
	"github.com/loft-sh/devspace/pkg/util/hash"
	"github.com/loft-sh/devspace/pkg/util/log"
	"io"
	"os"

	"github.com/pkg/errors"
)

// Dependency holds the dependency config and has an id
type Dependency struct {
	id           string
	name         string
	absolutePath string
	root         bool
	localConfig  config.Config

	children []types.Dependency

	dependencyConfig *latest.DependencyConfig
	dependencyCache  localcache.Cache

	kubeClient     kubectl.Client
	dockerClient   docker.Client
	registryClient pullsecrets.Client
}

// Implement Interface Methods

func (d *Dependency) ID() string { return d.id }

func (d *Dependency) Name() string { return d.name }

func (d *Dependency) Root() bool { return d.root }

func (d *Dependency) KubeClient() kubectl.Client { return d.kubeClient }

func (d *Dependency) Config() config.Config { return d.localConfig }

func (d *Dependency) Path() string { return d.absolutePath }

func (d *Dependency) DependencyConfig() *latest.DependencyConfig { return d.dependencyConfig }

func (d *Dependency) Children() []types.Dependency { return d.children }

func (d *Dependency) Command(command string, args []string) error {
	// Switch current working directory
	currentWorkingDirectory, err := d.prepare(true)
	if err != nil {
		return err
	} else if currentWorkingDirectory == "" {
		return nil
	}

	// Change back to original working directory
	defer func() { _ = os.Chdir(currentWorkingDirectory) }()
	return ExecuteCommand(d.localConfig.Config().Commands, command, args, os.Stdout, os.Stderr)
}

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
	defer func() { _ = os.Chdir(currentWorkingDirectory) }()

	// Check if image build is enabled
	_, err = d.buildImages(false, buildOptions, log)
	if err != nil {
		return err
	}
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
	defer func() { _ = os.Chdir(currentWorkingDirectory) }()

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
	if !skipDeploy {
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

	return nil
}

// Render renders the dependency
func (d *Dependency) Render(skipBuild bool, buildOptions *build.Options, out io.Writer, log log.Logger) error {
	// Switch current working directory
	currentWorkingDirectory, err := d.changeWorkingDirectory()
	if err != nil {
		return errors.Wrap(err, "getwd")
	}

	defer func() { _ = os.Chdir(currentWorkingDirectory) }()

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
	defer func() { _ = os.Chdir(currentWorkingDirectory) }()

	// Purge the deployments
	err = d.deployController.Purge(nil, log)
	if err != nil {
		log.Errorf("Error purging dependency %s: %v", d.Name(), err)
	}

	if d.generatedSaver != nil && d.localConfig != nil && d.localConfig.Generated() != nil {
		err = d.generatedSaver.Save(d.localConfig.Generated())
		if err != nil {
			log.Errorf("Error saving generated.yaml: %v", err)
		}
	}

	delete(d.dependencyCache.GetActive().Dependencies, d.id)
	return nil
}

func (d *Dependency) buildImages(skipBuild bool, buildOptions *build.Options, log log.Logger) (map[string]string, error) {
	var err error

	// Check if image build is enabled
	builtImages := make(map[string]string)
	if !skipBuild && !d.dependencyConfig.SkipBuild {
		// Build images
		builtImages, err = d.buildController.Build(buildOptions, log)
		if err != nil {
			return nil, err
		}

		// Save config if an image was built
		if len(builtImages) > 0 && d.generatedSaver != nil && d.localConfig != nil && d.localConfig.Generated() != nil {
			err := d.generatedSaver.Save(d.localConfig.Generated())
			if err != nil {
				return nil, errors.Errorf("Error saving generated config: %v", err)
			}
		}

		d.builtImages = builtImages
	}

	return builtImages, nil
}

func (d *Dependency) prepare(forceDependencies bool) (string, error) {
	// Check if we should redeploy
	directoryHash, err := hash.DirectoryExcludes(d.localPath, []string{".git", ".devspace"}, true)
	if err != nil {
		return "", errors.Wrap(err, "hash directory")
	}

	// Check if we skip the dependency deploy
	if !forceDependencies && directoryHash == d.dependencyCache.GetActive().Dependencies[d.id] {
		return "", nil
	}

	d.dependencyCache.GetActive().Dependencies[d.id] = directoryHash
	return d.changeWorkingDirectory()
}

func skipDependency(name string, skipDependencies []string) bool {
	for _, sd := range skipDependencies {
		if sd == name {
			return true
		}
	}
	return false
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
