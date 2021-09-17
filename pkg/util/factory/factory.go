package factory

import (
	"github.com/loft-sh/devspace/pkg/devspace/analyze"
	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/configure"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	dependencytypes "github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/helm"
	"github.com/loft-sh/devspace/pkg/devspace/helm/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/pullsecrets"
	"github.com/loft-sh/devspace/pkg/devspace/services"
	"github.com/loft-sh/devspace/pkg/util/kubeconfig"
	"github.com/loft-sh/devspace/pkg/util/log"
)

// Factory is the main interface for various client creations
type Factory interface {
	// NewConfigLoader creates a new config loader
	NewConfigLoader(configPath string) loader.ConfigLoader

	// NewConfigureManager creates a new configure manager
	NewConfigureManager(config *latest.Config, generated *generated.Config, log log.Logger) configure.Manager

	// NewKubeDefaultClient creates a new kube client
	NewKubeDefaultClient() (kubectl.Client, error)
	NewKubeClientFromContext(context, namespace string, switchContext bool) (kubectl.Client, error)
	NewKubeClientBySelect(allowPrivate bool, switchContext bool, log log.Logger) (kubectl.Client, error)

	// NewHelmClient creates a new helm client
	NewHelmClient(config *latest.Config, deployConfig *latest.DeploymentConfig, kubeClient kubectl.Client, tillerNamespace string, upgradeTiller bool, dryInit bool, log log.Logger) (types.Client, error)

	// NewDependencyManager creates a new dependency manager
	NewDependencyManager(config config.Config, client kubectl.Client, configOptions *loader.ConfigOptions, logger log.Logger) dependency.Manager

	// NewPullSecretClient creates a new pull secrets client
	NewPullSecretClient(config config.Config, dependencies []dependencytypes.Dependency, kubeClient kubectl.Client, dockerClient docker.Client, log log.Logger) pullsecrets.Client

	// NewDockerClient creates a new docker API client
	NewDockerClient(log log.Logger) (docker.Client, error)
	NewDockerClientWithMinikube(currentKubeContext string, preferMinikube bool, log log.Logger) (docker.Client, error)

	// NewServicesClient creates a new services client
	NewServicesClient(config config.Config, dependencies []dependencytypes.Dependency, kubeClient kubectl.Client, log log.Logger) services.Client

	// NewBuildController & NewDeployController
	NewBuildController(config config.Config, dependencies []dependencytypes.Dependency, client kubectl.Client) build.Controller
	NewDeployController(config config.Config, dependencies []dependencytypes.Dependency, client kubectl.Client) deploy.Controller

	// NewAnalyzer creates a new analyzer
	NewAnalyzer(client kubectl.Client, log log.Logger) analyze.Analyzer

	// NewKubeConfigLoader creates a new kube config loader
	NewKubeConfigLoader() kubeconfig.Loader

	// NewPluginManager creates a new plugin manager
	NewPluginManager(log log.Logger) plugin.Interface

	// GetLog retrieves the log instance
	GetLog() log.Logger
}

// DefaultFactoryImpl is the default factory implementation
type DefaultFactoryImpl struct{}

// DefaultFactory returns the default factory implementation
func DefaultFactory() Factory {
	return &DefaultFactoryImpl{}
}

// NewPluginManager creates a new plugin manager
func (f *DefaultFactoryImpl) NewPluginManager(log log.Logger) plugin.Interface {
	return plugin.NewClient(log)
}

// NewAnalyzer creates a new analyzer
func (f *DefaultFactoryImpl) NewAnalyzer(client kubectl.Client, log log.Logger) analyze.Analyzer {
	return analyze.NewAnalyzer(client, log)
}

// NewBuildController implements interface
func (f *DefaultFactoryImpl) NewBuildController(config config.Config, dependencies []dependencytypes.Dependency, client kubectl.Client) build.Controller {
	return build.NewController(config, dependencies, client)
}

// NewDeployController implements interface
func (f *DefaultFactoryImpl) NewDeployController(config config.Config, dependencies []dependencytypes.Dependency, client kubectl.Client) deploy.Controller {
	return deploy.NewController(config, dependencies, client)
}

// NewKubeConfigLoader implements interface
func (f *DefaultFactoryImpl) NewKubeConfigLoader() kubeconfig.Loader {
	return kubeconfig.NewLoader()
}

// GetLog implements interface
func (f *DefaultFactoryImpl) GetLog() log.Logger {
	return log.GetInstance()
}

// NewDependencyManager implements interface
func (f *DefaultFactoryImpl) NewDependencyManager(config config.Config, client kubectl.Client, configOptions *loader.ConfigOptions, logger log.Logger) dependency.Manager {
	return dependency.NewManager(config, client, configOptions, logger)
}

// NewPullSecretClient implements interface
func (f *DefaultFactoryImpl) NewPullSecretClient(config config.Config, dependencies []dependencytypes.Dependency, kubeClient kubectl.Client, dockerClient docker.Client, log log.Logger) pullsecrets.Client {
	return pullsecrets.NewClient(config, dependencies, kubeClient, dockerClient, log)
}

// NewConfigLoader implements interface
func (f *DefaultFactoryImpl) NewConfigLoader(configPath string) loader.ConfigLoader {
	return loader.NewConfigLoader(configPath)
}

// NewConfigureManager implements interface
func (f *DefaultFactoryImpl) NewConfigureManager(config *latest.Config, generated *generated.Config, log log.Logger) configure.Manager {
	return configure.NewManager(f, config, generated, log)
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
	client, err := kubectl.NewClientFromContext(context, namespace, switchContext, kubeLoader)
	if err != nil {
		return nil, err
	}

	plugin.SetPluginKubeContext(client.CurrentContext(), client.Namespace())
	return client, nil
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
func (f *DefaultFactoryImpl) NewServicesClient(config config.Config, dependencies []dependencytypes.Dependency, kubeClient kubectl.Client, log log.Logger) services.Client {
	return services.NewClient(config, dependencies, kubeClient, log)
}
