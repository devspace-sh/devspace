package factory

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/resume"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/dependency"
	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm/types"
	"github.com/devspace-cloud/devspace/pkg/devspace/hook"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/registry"
	"github.com/devspace-cloud/devspace/pkg/devspace/services"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// Factory is the main interface for various client creations
type Factory interface {
	// Config Loader
	NewConfigLoader(options *loader.ConfigOptions, log log.Logger) loader.ConfigLoader

	// Kubernetes Clients
	NewKubeDefaultClient() (kubectl.Client, error)
	NewKubeClientFromContext(context, namespace string, switchContext bool) (kubectl.Client, error)
	NewKubeClientBySelect(allowPrivate bool, switchContext bool, log log.Logger) (kubectl.Client, error)

	// Helm
	NewHelmClient(config *latest.Config, deployConfig *latest.DeploymentConfig, kubeClient kubectl.Client, tillerNamespace string, upgradeTiller bool, log log.Logger) (types.Client, error)

	// Dependencies
	NewDependencyManager(config *latest.Config, cache *generated.Config, client kubectl.Client, allowCyclic bool, configOptions *loader.ConfigOptions, logger log.Logger) (dependency.Manager, error)

	// Hooks
	NewHookExecutor(config *latest.Config) hook.Executer

	// Pull secrets client
	NewPullSecretClient(config *latest.Config, kubeClient kubectl.Client, dockerClient docker.Client, log log.Logger) registry.Client

	// Docker
	NewDockerClient(log log.Logger) (docker.Client, error)
	NewDockerClientWithMinikube(currentKubeContext string, preferMinikube bool, log log.Logger) (docker.Client, error)

	// Services
	NewServicesClient(config *latest.Config, generated *generated.Config, kubeClient kubectl.Client, selectorParameter *targetselector.SelectorParameter, log log.Logger) services.Client

	// Cloud
	GetProvider(useProviderName string, log log.Logger) (cloud.Provider, error)
	GetProviderWithOptions(useProviderName, key string, relogin bool, loader config.Loader, log log.Logger) (cloud.Provider, error)
	NewSpaceResumer(kubeClient kubectl.Client, log log.Logger) resume.SpaceResumer

	// Log
	GetLog() log.Logger
}

type factory struct{}

// DefaultFactory returns the default factory implementation
func DefaultFactory() Factory {
	return &factory{}
}

func (f *factory) GetLog() log.Logger {
	return log.GetInstance()
}

func (f *factory) NewHookExecutor(config *latest.Config) hook.Executer {
	return hook.NewExecuter(config)
}

func (f *factory) NewDependencyManager(config *latest.Config, cache *generated.Config, client kubectl.Client, allowCyclic bool, configOptions *loader.ConfigOptions, logger log.Logger) (dependency.Manager, error) {
	return dependency.NewManager(config, cache, client, allowCyclic, configOptions, logger)
}

func (f *factory) NewPullSecretClient(config *latest.Config, kubeClient kubectl.Client, dockerClient docker.Client, log log.Logger) registry.Client {
	return registry.NewClient(config, kubeClient, dockerClient, log)
}

func (f *factory) NewConfigLoader(options *loader.ConfigOptions, log log.Logger) loader.ConfigLoader {
	return loader.NewConfigLoader(options, log)
}

func (f *factory) NewDockerClient(log log.Logger) (docker.Client, error) {
	return docker.NewClient(log)
}

func (f *factory) NewDockerClientWithMinikube(currentKubeContext string, preferMinikube bool, log log.Logger) (docker.Client, error) {
	return docker.NewClientWithMinikube(currentKubeContext, preferMinikube, log)
}

func (f *factory) NewKubeDefaultClient() (kubectl.Client, error) {
	return kubectl.NewDefaultClient()
}

func (f *factory) NewKubeClientFromContext(context, namespace string, switchContext bool) (kubectl.Client, error) {
	return kubectl.NewClientFromContext(context, namespace, switchContext)
}

func (f *factory) NewKubeClientBySelect(allowPrivate bool, switchContext bool, log log.Logger) (kubectl.Client, error) {
	return kubectl.NewClientBySelect(allowPrivate, switchContext, log)
}

func (f *factory) NewHelmClient(config *latest.Config, deployConfig *latest.DeploymentConfig, kubeClient kubectl.Client, tillerNamespace string, upgradeTiller bool, log log.Logger) (types.Client, error) {
	return helm.NewClient(config, deployConfig, kubeClient, tillerNamespace, upgradeTiller, log)
}

func (f *factory) NewServicesClient(config *latest.Config, generated *generated.Config, kubeClient kubectl.Client, selectorParameter *targetselector.SelectorParameter, log log.Logger) services.Client {
	return services.NewClient(config, generated, kubeClient, selectorParameter, log)
}

func (f *factory) GetProvider(useProviderName string, log log.Logger) (cloud.Provider, error) {
	return cloud.GetProvider(useProviderName, log)
}

func (f *factory) GetProviderWithOptions(useProviderName, key string, relogin bool, loader config.Loader, log log.Logger) (cloud.Provider, error) {
	return cloud.GetProviderWithOptions(useProviderName, key, relogin, loader, log)
}

func (f *factory) NewSpaceResumer(kubeClient kubectl.Client, log log.Logger) resume.SpaceResumer {
	return resume.NewSpaceResumer(kubeClient, log)
}
