package factory

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/analyze"
	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/configure"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/helm"
	"github.com/loft-sh/devspace/pkg/devspace/helm/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/kubeconfig"
	"github.com/loft-sh/devspace/pkg/util/log"
)

// Factory is the main interface for various client creations
type Factory interface {
	// NewConfigLoader creates a new config loader
	NewConfigLoader(configPath string) (loader.ConfigLoader, error)

	// NewConfigureManager creates a new configure manager
	NewConfigureManager(config *latest.Config, generated localcache.Cache, log log.Logger) configure.Manager

	// NewKubeDefaultClient creates a new kube client
	NewKubeDefaultClient() (kubectl.Client, error)
	NewKubeClientFromContext(context, namespace string) (kubectl.Client, error)

	// NewHelmClient creates a new helm client
	NewHelmClient(log log.Logger) (types.Client, error)

	// NewDependencyManager creates a new dependency manager
	NewDependencyManager(ctx devspacecontext.Context, configOptions *loader.ConfigOptions) dependency.Manager

	// NewDockerClient creates a new docker API client
	NewDockerClient(ctx context.Context, log log.Logger) (docker.Client, error)
	NewDockerClientWithMinikube(ctx context.Context, client kubectl.Client, preferMinikube bool, log log.Logger) (docker.Client, error)

	// NewBuildController & NewDeployController
	NewBuildController() build.Controller
	NewDeployController() deploy.Controller

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
func (f *DefaultFactoryImpl) NewBuildController() build.Controller {
	return build.NewController()
}

// NewDeployController implements interface
func (f *DefaultFactoryImpl) NewDeployController() deploy.Controller {
	return deploy.NewController()
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
func (f *DefaultFactoryImpl) NewDependencyManager(ctx devspacecontext.Context, configOptions *loader.ConfigOptions) dependency.Manager {
	return dependency.NewManager(ctx, configOptions)
}

// NewConfigLoader implements interface
func (f *DefaultFactoryImpl) NewConfigLoader(configPath string) (loader.ConfigLoader, error) {
	return loader.NewConfigLoader(configPath)
}

// NewConfigureManager implements interface
func (f *DefaultFactoryImpl) NewConfigureManager(config *latest.Config, generated localcache.Cache, log log.Logger) configure.Manager {
	return configure.NewManager(f, config, generated, log)
}

// NewDockerClient implements interface
func (f *DefaultFactoryImpl) NewDockerClient(ctx context.Context, log log.Logger) (docker.Client, error) {
	return docker.NewClient(ctx, log)
}

// NewDockerClientWithMinikube implements interface
func (f *DefaultFactoryImpl) NewDockerClientWithMinikube(ctx context.Context, kubectlClient kubectl.Client, preferMinikube bool, log log.Logger) (docker.Client, error) {
	return docker.NewClientWithMinikube(ctx, kubectlClient, preferMinikube, log)
}

// NewKubeDefaultClient implements interface
func (f *DefaultFactoryImpl) NewKubeDefaultClient() (kubectl.Client, error) {
	return kubectl.NewDefaultClient()
}

// NewKubeClientFromContext implements interface
func (f *DefaultFactoryImpl) NewKubeClientFromContext(context, namespace string) (kubectl.Client, error) {
	kubeLoader := f.NewKubeConfigLoader()
	client, err := kubectl.NewClientFromContext(context, namespace, false, kubeLoader)
	if err != nil {
		return nil, err
	}

	plugin.SetPluginKubeContext(client.CurrentContext(), client.Namespace())
	return client, nil
}

// NewHelmClient implements interface
func (f *DefaultFactoryImpl) NewHelmClient(log log.Logger) (types.Client, error) {
	return helm.NewClient(log)
}
