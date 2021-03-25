package factory

import (
	"github.com/loft-sh/devspace/pkg/devspace/analyze"
	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/configure"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/helm"
	"github.com/loft-sh/devspace/pkg/devspace/helm/types"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/pullsecrets"
	"github.com/loft-sh/devspace/pkg/devspace/services"
	"github.com/loft-sh/devspace/pkg/util/kubeconfig"
	"github.com/loft-sh/devspace/pkg/util/log"
)

// Factory is the main interface for various client creations
type Factory interface {
	// Config Loader
	NewConfigLoader(configPath string) loader.ConfigLoader

	// ConfigureManager
	NewConfigureManager(config *latest.Config, log log.Logger) configure.Manager

	// Kubernetes Clients
	NewKubeDefaultClient() (kubectl.Client, error)
	NewKubeClientFromContext(context, namespace string, switchContext bool) (kubectl.Client, error)
	NewKubeClientBySelect(allowPrivate bool, switchContext bool, log log.Logger) (kubectl.Client, error)

	// Helm
	NewHelmClient(config *latest.Config, deployConfig *latest.DeploymentConfig, kubeClient kubectl.Client, tillerNamespace string, upgradeTiller bool, dryInit bool, log log.Logger) (types.Client, error)

	// Dependencies
	NewDependencyManager(config *latest.Config, cache *generated.Config, client kubectl.Client, allowCyclic bool, configOptions *loader.ConfigOptions, logger log.Logger) (dependency.Manager, error)

	// Hooks
	NewHookExecutor(config *latest.Config) hook.Executer

	// Pull secrets client
	NewPullSecretClient(config *latest.Config, cache *generated.CacheConfig, kubeClient kubectl.Client, dockerClient docker.Client, log log.Logger) pullsecrets.Client

	// Docker
	NewDockerClient(log log.Logger) (docker.Client, error)
	NewDockerClientWithMinikube(currentKubeContext string, preferMinikube bool, log log.Logger) (docker.Client, error)

	// Services
	NewServicesClient(config *latest.Config, generated *generated.Config, kubeClient kubectl.Client, log log.Logger) services.Client

	// Build & Deploy
	NewBuildController(config *latest.Config, cache *generated.CacheConfig, client kubectl.Client) build.Controller
	NewDeployController(config *latest.Config, cache *generated.CacheConfig, client kubectl.Client) deploy.Controller

	// Analyzer
	NewAnalyzer(client kubectl.Client, log log.Logger) analyze.Analyzer

	// Kubeconfig
	NewKubeConfigLoader() kubeconfig.Loader

	// Plugin
	NewPluginManager(log log.Logger) plugin.Interface

	// Log
	GetLog() log.Logger
}

// DefaultFactoryImpl is the default factory implementation
type DefaultFactoryImpl struct{}

// DefaultFactory returns the default factory implementation
func DefaultFactory() Factory {
	return &DefaultFactoryImpl{}
}

// NewPluginsManager creates a new plugin manager
func (f *DefaultFactoryImpl) NewPluginManager(log log.Logger) plugin.Interface {
	return plugin.NewClient(log)
}

// NewAnalyzer creates a new analyzer
func (f *DefaultFactoryImpl) NewAnalyzer(client kubectl.Client, log log.Logger) analyze.Analyzer {
	return analyze.NewAnalyzer(client, log)
}

// NewBuildController implements interface
func (f *DefaultFactoryImpl) NewBuildController(config *latest.Config, cache *generated.CacheConfig, client kubectl.Client) build.Controller {
	return build.NewController(config, cache, client)
}

// NewDeployController implements interface
func (f *DefaultFactoryImpl) NewDeployController(config *latest.Config, cache *generated.CacheConfig, client kubectl.Client) deploy.Controller {
	return deploy.NewController(config, cache, client)
}

// NewKubeConfigLoader implements interface
func (f *DefaultFactoryImpl) NewKubeConfigLoader() kubeconfig.Loader {
	return kubeconfig.NewLoader()
}

// GetLog implements interface
func (f *DefaultFactoryImpl) GetLog() log.Logger {
	return log.GetInstance()
}

// NewHookExecutor implements interface
func (f *DefaultFactoryImpl) NewHookExecutor(config *latest.Config) hook.Executer {
	return hook.NewExecuter(config)
}

// NewDependencyManager implements interface
func (f *DefaultFactoryImpl) NewDependencyManager(config *latest.Config, cache *generated.Config, client kubectl.Client, allowCyclic bool, configOptions *loader.ConfigOptions, logger log.Logger) (dependency.Manager, error) {
	return dependency.NewManager(config, cache, client, allowCyclic, configOptions, logger)
}

// NewPullSecretClient implements interface
func (f *DefaultFactoryImpl) NewPullSecretClient(config *latest.Config, cache *generated.CacheConfig, kubeClient kubectl.Client, dockerClient docker.Client, log log.Logger) pullsecrets.Client {
	return pullsecrets.NewClient(config, cache, kubeClient, dockerClient, log)
}

// NewConfigLoader implements interface
func (f *DefaultFactoryImpl) NewConfigLoader(configPath string) loader.ConfigLoader {
	return loader.NewConfigLoader(configPath)
}

// NewConfigureManager implements interface
func (f *DefaultFactoryImpl) NewConfigureManager(config *latest.Config, log log.Logger) configure.Manager {
	return configure.NewManager(f, config, log)
}

// NewDockerClient implements interface
func (f *DefaultFactoryImpl) NewDockerClient(log log.Logger) (docker.Client, error) {
	return docker.NewClient(log)
}

// NewDockerClientWithMinikube implements interface
func (f *DefaultFactoryImpl) NewDockerClientWithMinikube(currentKubeContext string, preferMinikube bool, log log.Logger) (docker.Client, error) {
	return docker.NewClientWithMinikube(currentKubeContext, preferMinikube, log)
}

// NewKubeDefaultClient implements interface
func (f *DefaultFactoryImpl) NewKubeDefaultClient() (kubectl.Client, error) {
	return kubectl.NewDefaultClient()
}

// NewKubeClientFromContext implements interface
func (f *DefaultFactoryImpl) NewKubeClientFromContext(context, namespace string, switchContext bool) (kubectl.Client, error) {
	kubeLoader := f.NewKubeConfigLoader()
	return kubectl.NewClientFromContext(context, namespace, switchContext, kubeLoader)
}

// NewKubeClientBySelect implements interface
func (f *DefaultFactoryImpl) NewKubeClientBySelect(allowPrivate bool, switchContext bool, log log.Logger) (kubectl.Client, error) {
	kubeLoader := f.NewKubeConfigLoader()
	return kubectl.NewClientBySelect(allowPrivate, switchContext, kubeLoader, log)
}

// NewHelmClient implements interface
func (f *DefaultFactoryImpl) NewHelmClient(config *latest.Config, deployConfig *latest.DeploymentConfig, kubeClient kubectl.Client, tillerNamespace string, upgradeTiller bool, dryInit bool, log log.Logger) (types.Client, error) {
	return helm.NewClient(config, deployConfig, kubeClient, tillerNamespace, upgradeTiller, dryInit, log)
}

// NewServicesClient implements interface
func (f *DefaultFactoryImpl) NewServicesClient(config *latest.Config, generated *generated.Config, kubeClient kubectl.Client, log log.Logger) services.Client {
	return services.NewClient(config, generated, kubeClient, log)
}
