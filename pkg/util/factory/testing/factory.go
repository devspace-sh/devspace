package testing

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
	"github.com/loft-sh/devspace/pkg/devspace/helm/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/pullsecrets"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/kubeconfig"
	"github.com/loft-sh/devspace/pkg/util/log"
)

// Make sure the test interface implements the interface
var _ factory.Factory = &Factory{}

// Factory implements the Factory interface
type Factory struct {
	Analyzer          analyze.Analyzer
	BuildController   build.Controller
	DeployController  deploy.Controller
	KubeconfigLoader  kubeconfig.Loader
	Log               log.Logger
	DependencyManager dependency.Manager
	PullSecretClient  pullsecrets.Client
	ConfigLoader      loader.ConfigLoader
	ConfigureManager  configure.Manager
	DockerClient      docker.Client
	KubeClient        kubectl.Client
	HelmClient        types.Client
	PluginClient      plugin.Interface
}

// NewPluginsManager creates a new plugin manager
func (f *Factory) NewPluginManager(log log.Logger) plugin.Interface {
	return f.PluginClient
}

// NewAnalyzer creates a new analyzer
func (f *Factory) NewAnalyzer(client kubectl.Client, log log.Logger) analyze.Analyzer {
	return f.Analyzer
}

// NewBuildController implements interface
func (f *Factory) NewBuildController() build.Controller {
	return f.BuildController
}

// NewDeployController implements interface
func (f *Factory) NewDeployController() deploy.Controller {
	return f.DeployController
}

// NewKubeConfigLoader implements interface
func (f *Factory) NewKubeConfigLoader() kubeconfig.Loader {
	return f.KubeconfigLoader
}

// GetLog implements interface
func (f *Factory) GetLog() log.Logger {
	return f.Log
}

// NewDependencyManager implements interface
func (f *Factory) NewDependencyManager(ctx devspacecontext.Context, configOptions *loader.ConfigOptions) dependency.Manager {
	return f.DependencyManager
}

// NewPullSecretClient implements interface
func (f *Factory) NewPullSecretClient(dockerClient docker.Client) pullsecrets.Client {
	return f.PullSecretClient
}

// NewConfigLoader implements interface
func (f *Factory) NewConfigLoader(configPath string) (loader.ConfigLoader, error) {
	return f.ConfigLoader, nil
}

// NewConfigureManager implements interface
func (f *Factory) NewConfigureManager(config *latest.Config, generated localcache.Cache, log log.Logger) configure.Manager {
	return f.ConfigureManager
}

// NewDockerClient implements interface
func (f *Factory) NewDockerClient(ctx context.Context, log log.Logger) (docker.Client, error) {
	return f.DockerClient, nil
}

// NewDockerClientWithMinikube implements interface
func (f *Factory) NewDockerClientWithMinikube(ctx context.Context, kubeClient kubectl.Client, preferMinikube bool, log log.Logger) (docker.Client, error) {
	return f.DockerClient, nil
}

// NewKubeDefaultClient implements interface
func (f *Factory) NewKubeDefaultClient() (kubectl.Client, error) {
	return f.KubeClient, nil
}

// NewKubeClientFromContext implements interface
func (f *Factory) NewKubeClientFromContext(context, namespace string) (kubectl.Client, error) {
	return f.KubeClient, nil
}

// NewHelmClient implements interface
func (f *Factory) NewHelmClient(log log.Logger) (types.Client, error) {
	return f.HelmClient, nil
}
