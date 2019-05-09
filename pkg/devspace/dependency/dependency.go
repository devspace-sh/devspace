package dependency

import (
	"fmt"
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/build"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	deploy "github.com/devspace-cloud/devspace/pkg/devspace/deploy/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/registry"
	"github.com/devspace-cloud/devspace/pkg/util/hash"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

// DeployAll will deploy all dependencies if there are any
func DeployAll(config *latest.Config, cache *generated.CacheConfig, allowCyclic, updateDependencies, createPullSecrets, forceDeployDependencies, forceBuild, forceDeploy bool, log log.Logger) error {
	if config == nil || config.Dependencies == nil || len(*config.Dependencies) == 0 {
		return nil
	}

	// Create a new dependency resolver
	resolver, err := NewResolver(config, cache, allowCyclic, log)
	if err != nil {
		return errors.Wrap(err, "new resolver")
	}

	// Resolve all dependencies
	dependencies, err := resolver.Resolve(*config.Dependencies, updateDependencies)
	if err != nil {
		return err
	}

	// Deploy all dependencies
	for _, dependency := range dependencies {
		err := dependency.Deploy(createPullSecrets, forceDeployDependencies, forceBuild, forceDeploy, log)
		if err != nil {
			return errors.Wrap(err, "deploy dependency "+dependency.ID)
		}
	}

	return nil
}

// PurgeAll purges all dependencies in reverse order
func PurgeAll(config *latest.Config, cache *generated.CacheConfig, allowCyclic bool, log log.Logger) error {
	if config == nil || config.Dependencies == nil || len(*config.Dependencies) == 0 {
		return nil
	}

	// Create a new dependency resolver
	resolver, err := NewResolver(config, cache, allowCyclic, log)
	if err != nil {
		return err
	}

	// Resolve all dependencies
	dependencies, err := resolver.Resolve(*config.Dependencies, false)
	if err != nil {
		return errors.Wrap(err, "resolve dependencies")
	}

	// Purge all dependencies
	for i := len(dependencies) - 1; i >= 0; i-- {
		dependency := dependencies[i]

		err := dependency.Purge(log)
		if err != nil {
			return errors.Wrap(err, "purge dependency "+dependency.ID)
		}
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
	DependencyCache  *generated.CacheConfig
}

// Deploy deploys the dependency if necessary
func (d *Dependency) Deploy(createPullSecrets bool, forceDependencies, forceBuild, forceDeploy bool, log log.Logger) error {
	// Check if we should redeploy
	directoryHash, err := hash.DirectoryExcludes(d.LocalPath, []string{".git", ".devspace"}, true)
	if err != nil {
		return errors.Wrap(err, "hash directory")
	}

	// Check if we skip the dependency deploy
	if forceDependencies == false && directoryHash == d.DependencyCache.Dependencies[d.ID] {
		return nil
	}

	d.DependencyCache.Dependencies[d.ID] = directoryHash

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

	log.StartWait("Deploy dependency " + d.ID)
	defer log.StopWait()

	// Create kubectl client
	client, err := kubectl.NewClient(d.Config)
	if err != nil {
		return fmt.Errorf("Unable to create new kubectl client: %v", err)
	}

	// Create namespace if necessary
	err = kubectl.EnsureDefaultNamespace(d.Config, client, log)
	if err != nil {
		return fmt.Errorf("Unable to create namespace: %v", err)
	}

	// Create the image pull secrets and add them to the default service account
	if createPullSecrets {
		// Create docker client
		dockerClient, err := docker.NewClient(d.Config, false)

		// Create pull secrets and private registry if necessary
		err = registry.CreatePullSecrets(d.Config, dockerClient, client, log)
		if err != nil {
			return err
		}
	}

	log.StopWait()

	// Build images
	builtImages, err := build.All(d.Config, d.GeneratedConfig.GetActive(), client, false, forceBuild, false, log)
	if err != nil {
		return err
	}

	// Save config if an image was built
	if len(builtImages) > 0 {
		err := generated.SaveConfig(d.GeneratedConfig)
		if err != nil {
			return fmt.Errorf("Error saving generated config: %v", err)
		}
	}

	// Deploy all defined deployments
	err = deploy.All(d.Config, d.GeneratedConfig.GetActive(), client, false, forceDeploy, builtImages, log)
	if err != nil {
		return err
	}

	// Save Config
	err = generated.SaveConfig(d.GeneratedConfig)
	if err != nil {
		return fmt.Errorf("Error saving generated config: %v", err)
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

	kubectl, err := kubectl.NewClient(d.Config)
	if err != nil {
		return fmt.Errorf("Unable to create new kubectl client: %v", err)
	}

	// Purge the deployments
	deploy.PurgeDeployments(d.Config, d.GeneratedConfig.GetActive(), kubectl, nil)

	err = generated.SaveConfig(d.GeneratedConfig)
	if err != nil {
		log.Errorf("Error saving generated.yaml: %v", err)
	}

	delete(d.DependencyCache.Dependencies, d.ID)
	log.Donef("Purged dependency %s", d.ID)
	return nil
}
