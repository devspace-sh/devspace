package dependency

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/build"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	deploy "github.com/devspace-cloud/devspace/pkg/devspace/deploy/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/registry"
	"github.com/devspace-cloud/devspace/pkg/util/hash"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// UpdateAll will update all dependencies if there are any
func UpdateAll(config *latest.Config, cache *generated.Config, allowCyclic bool, log log.Logger) error {
	if config == nil || config.Dependencies == nil || len(*config.Dependencies) == 0 {
		return nil
	}

	log.StartWait("Update dependencies")
	defer log.StopWait()

	// Create a new dependency resolver
	resolver, err := NewResolver(config, cache, allowCyclic, log)
	if err != nil {
		return errors.Wrap(err, "new resolver")
	}

	// Resolve all dependencies
	_, err = resolver.Resolve(context.Background(), *config.Dependencies, true)
	if err != nil {
		if _, ok := err.(*CyclicError); ok {
			return fmt.Errorf("%v.\n To allow cyclic dependencies run with the '%s' flag", err, ansi.Color("--allow-cyclic", "white+b"))
		}

		return err
	}

	return nil
}

// BuildAll will build all dependencies if there are any
func BuildAll(config *latest.Config, cache *generated.Config, allowCyclic, updateDependencies, skipPush, forceDeployDependencies, forceBuild bool, logger log.Logger) error {
	if config == nil || config.Dependencies == nil || len(*config.Dependencies) == 0 {
		return nil
	}

	// Create a new dependency resolver
	resolver, err := NewResolver(config, cache, allowCyclic, logger)
	if err != nil {
		return errors.Wrap(err, "new resolver")
	}

	// Resolve all dependencies
	dependencies, err := resolver.Resolve(context.Background(), *config.Dependencies, updateDependencies)
	if err != nil {
		if _, ok := err.(*CyclicError); ok {
			return fmt.Errorf("%v.\n To allow cyclic dependencies run with the '%s' flag", err, ansi.Color("--allow-cyclic", "white+b"))
		}

		return err
	}

	defer logger.StopWait()

	// Deploy all dependencies
	for i := 0; i < len(dependencies); i++ {
		dependency := dependencies[i]

		logger.StartWait(fmt.Sprintf("Building dependency %d of %d: %s", i+1, len(dependencies), dependency.ID))
		buff := &bytes.Buffer{}
		streamLog := log.NewStreamLogger(buff, logrus.InfoLevel)

		err := dependency.Build(skipPush, forceDeployDependencies, forceBuild, streamLog)
		if err != nil {
			return fmt.Errorf("Error building dependency %s: %s %v", dependency.ID, buff.String(), err)
		}

		logger.Donef("Built dependency %s", dependency.ID)
	}

	logger.StopWait()
	logger.Donef("Successfully built %d dependencies", len(dependencies))

	return nil
}

// DeployAll will deploy all dependencies if there are any
func DeployAll(config *latest.Config, cache *generated.Config, client *kubectl.Client, allowCyclic, updateDependencies, skipPush, forceDeployDependencies, skipBuild, forceBuild, forceDeploy bool, logger log.Logger) error {
	if config == nil || config.Dependencies == nil || len(*config.Dependencies) == 0 {
		return nil
	}

	// Create a new dependency resolver
	resolver, err := NewResolver(config, cache, allowCyclic, logger)
	if err != nil {
		return errors.Wrap(err, "new resolver")
	}

	// Resolve all dependencies
	dependencies, err := resolver.Resolve(context.WithValue(context.Background(), constants.KubeContextKey, client.CurrentContext), *config.Dependencies, updateDependencies)
	if err != nil {
		if _, ok := err.(*CyclicError); ok {
			return fmt.Errorf("%v.\n To allow cyclic dependencies run with the '%s' flag", err, ansi.Color("--allow-cyclic", "white+b"))
		}

		return err
	}

	defer logger.StopWait()

	// Deploy all dependencies
	for i := 0; i < len(dependencies); i++ {
		dependency := dependencies[i]

		logger.StartWait(fmt.Sprintf("Deploying dependency %d of %d: %s", i+1, len(dependencies), dependency.ID))
		buff := &bytes.Buffer{}
		streamLog := log.NewStreamLogger(buff, logrus.InfoLevel)

		err := dependency.Deploy(client, skipPush, forceDeployDependencies, skipBuild, forceBuild, forceDeploy, streamLog)
		if err != nil {
			return fmt.Errorf("Error deploying dependency %s: %s %v", dependency.ID, buff.String(), err)
		}

		// Prettify path if its a path deployment
		logger.Donef("Deployed dependency %s", dependency.ID)
	}

	logger.StopWait()
	logger.Donef("Successfully deployed %d dependencies", len(dependencies))

	return nil
}

// PurgeAll purges all dependencies in reverse order
func PurgeAll(config *latest.Config, cache *generated.Config, client *kubectl.Client, allowCyclic bool, logger log.Logger) error {
	if config == nil || config.Dependencies == nil || len(*config.Dependencies) == 0 {
		return nil
	}

	// Create a new dependency resolver
	resolver, err := NewResolver(config, cache, allowCyclic, logger)
	if err != nil {
		return err
	}

	// Resolve all dependencies
	dependencies, err := resolver.Resolve(context.WithValue(context.Background(), constants.KubeContextKey, client.CurrentContext), *config.Dependencies, false)
	if err != nil {
		if _, ok := err.(*CyclicError); ok {
			return fmt.Errorf("%v.\n To allow cyclic dependencies run with the '%s' flag", err, ansi.Color("--allow-cyclic", "white+b"))
		}

		return errors.Wrap(err, "resolve dependencies")
	}

	defer logger.StopWait()

	// Purge all dependencies
	for i := len(dependencies) - 1; i >= 0; i-- {
		logger.StartWait(fmt.Sprintf("Purging %d dependencies", i+1))
		dependency := dependencies[i]

		buff := &bytes.Buffer{}
		streamLog := log.NewStreamLogger(buff, logrus.InfoLevel)

		err := dependency.Purge(client, streamLog)
		if err != nil {
			return fmt.Errorf("Error deploying dependency %s: %s %v", dependency.ID, buff.String(), err)
		}

		logger.Donef("Purged dependency %s", dependency.ID)
	}

	logger.StopWait()
	logger.Donef("Successfully purged %d dependencies", len(dependencies))

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
		builtImages, err = build.All(d.Config, d.GeneratedConfig.GetActive(), nil, skipPush, false, forceBuild, false, log)
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
	}

	log.Donef("Built dependency %s", d.ID)
	return nil
}

// Deploy deploys the dependency if necessary
func (d *Dependency) Deploy(client *kubectl.Client, skipPush, forceDependencies, skipBuild, forceBuild, forceDeploy bool, log log.Logger) error {
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

	// Recreate client if necessary
	if d.DependencyConfig.Namespace != nil && *d.DependencyConfig.Namespace != "" {
		client, err = kubectl.NewClientFromContext(client.CurrentContext, *d.DependencyConfig.Namespace, false)
		if err != nil {
			return errors.Wrap(err, "create new client")
		}
	}

	// Create namespace if necessary
	err = client.EnsureDefaultNamespace(log)
	if err != nil {
		return fmt.Errorf("Unable to create namespace: %v", err)
	}

	// Create docker client
	dockerClient, err := docker.NewClient(log)
	if err != nil {
		return errors.Wrap(err, "create docker client")
	}

	// Create pull secrets and private registry if necessary
	err = registry.CreatePullSecrets(d.Config, client, dockerClient, log)
	if err != nil {
		return err
	}

	// Check if image build is enabled
	builtImages := make(map[string]string)
	if skipBuild == false && (d.DependencyConfig.SkipBuild == nil || *d.DependencyConfig.SkipBuild == false) {
		// Build images
		builtImages, err = build.All(d.Config, d.GeneratedConfig.GetActive(), client, skipPush, false, forceBuild, false, log)
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
	}

	// Deploy all defined deployments
	err = deploy.All(d.Config, d.GeneratedConfig.GetActive(), client, false, forceDeploy, builtImages, nil, log)
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
func (d *Dependency) Purge(client *kubectl.Client, log log.Logger) error {
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

	// Recreate client if necessary
	if d.DependencyConfig.Namespace != nil && *d.DependencyConfig.Namespace != "" {
		client, err = kubectl.NewClientFromContext(client.CurrentContext, *d.DependencyConfig.Namespace, false)
		if err != nil {
			return errors.Wrap(err, "create new client")
		}
	}

	// Purge the deployments
	deploy.PurgeDeployments(d.Config, d.GeneratedConfig.GetActive(), client, nil, log)

	err = generated.SaveConfig(d.GeneratedConfig)
	if err != nil {
		log.Errorf("Error saving generated.yaml: %v", err)
	}

	delete(d.DependencyCache.GetActive().Dependencies, d.ID)
	log.Donef("Purged dependency %s", d.ID)
	return nil
}
