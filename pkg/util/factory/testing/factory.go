package testing

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/analyze"
	"github.com/devspace-cloud/devspace/pkg/devspace/build"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/resume"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/configure"
	"github.com/devspace-cloud/devspace/pkg/devspace/dependency"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy"
	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm/types"
	"github.com/devspace-cloud/devspace/pkg/devspace/hook"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/registry"
	"github.com/devspace-cloud/devspace/pkg/devspace/services"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// Make sure the test interface implements the interface
var _ factory.Factory = &Factory{}

// Factory implements the Factory interface
type Factory struct {
	Analyzer          analyze.Analyzer
	CloudConfigLoader config.Loader
	BuildController   build.Controller
	DeployController  deploy.Controller
	KubeconfigLoader  kubeconfig.Loader
	Log               log.Logger
	HookExecutor      hook.Executer
	DependencyManager dependency.Manager
	PullSecretClient  registry.Client
	ConfigLoader      loader.ConfigLoader
	ConfigureManager  configure.Manager
	DockerClient      docker.Client
	KubeClient        kubectl.Client
	HelmClient        types.Client
	ServicesClient    services.Client
	Provider          cloud.Provider
	Resumer           resume.SpaceResumer
}

// NewAnalyzer creates a new analyzer
func (f *Factory) NewAnalyzer(client kubectl.Client, log log.Logger) analyze.Analyzer {
	return f.Analyzer
}

// NewCloudConfigLoader creates a new cloud config loader
func (f *Factory) NewCloudConfigLoader() config.Loader {
	return f.CloudConfigLoader
}

// NewBuildController implements interface
func (f *Factory) NewBuildController(config *latest.Config, cache *generated.CacheConfig, client kubectl.Client) build.Controller {
	return f.BuildController
}

// NewDeployController implements interface
func (f *Factory) NewDeployController(config *latest.Config, cache *generated.CacheConfig, client kubectl.Client) deploy.Controller {
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

// NewHookExecutor implements interface
func (f *Factory) NewHookExecutor(config *latest.Config) hook.Executer {
	return f.HookExecutor
}

// NewDependencyManager implements interface
func (f *Factory) NewDependencyManager(config *latest.Config, cache *generated.Config, client kubectl.Client, allowCyclic bool, configOptions *loader.ConfigOptions, logger log.Logger) (dependency.Manager, error) {
	return f.DependencyManager, nil
}

// NewPullSecretClient implements interface
func (f *Factory) NewPullSecretClient(config *latest.Config, kubeClient kubectl.Client, dockerClient docker.Client, log log.Logger) registry.Client {
	return f.PullSecretClient
}

// NewConfigLoader implements interface
func (f *Factory) NewConfigLoader(options *loader.ConfigOptions, log log.Logger) loader.ConfigLoader {
	return f.ConfigLoader
}

// NewConfigureManager implements interface
func (f *Factory) NewConfigureManager(config *latest.Config, log log.Logger) configure.Manager {
	return f.ConfigureManager
}

// NewDockerClient implements interface
func (f *Factory) NewDockerClient(log log.Logger) (docker.Client, error) {
	return f.DockerClient, nil
}

// NewDockerClientWithMinikube implements interface
func (f *Factory) NewDockerClientWithMinikube(currentKubeContext string, preferMinikube bool, log log.Logger) (docker.Client, error) {
	return f.DockerClient, nil
}

// NewKubeDefaultClient implements interface
func (f *Factory) NewKubeDefaultClient() (kubectl.Client, error) {
	return f.KubeClient, nil
}

// NewKubeClientFromContext implements interface
func (f *Factory) NewKubeClientFromContext(context, namespace string, switchContext bool) (kubectl.Client, error) {
	return f.KubeClient, nil
}

// NewKubeClientBySelect implements interface
func (f *Factory) NewKubeClientBySelect(allowPrivate bool, switchContext bool, log log.Logger) (kubectl.Client, error) {
	return f.KubeClient, nil
}

// NewHelmClient implements interface
func (f *Factory) NewHelmClient(config *latest.Config, deployConfig *latest.DeploymentConfig, kubeClient kubectl.Client, tillerNamespace string, upgradeTiller bool, dryInit bool, log log.Logger) (types.Client, error) {
	return f.HelmClient, nil
}

// NewServicesClient implements interface
func (f *Factory) NewServicesClient(config *latest.Config, generated *generated.Config, kubeClient kubectl.Client, selectorParameter *targetselector.SelectorParameter, log log.Logger) services.Client {
	return f.ServicesClient
}

// GetProvider implements interface
func (f *Factory) GetProvider(useProviderName string, log log.Logger) (cloud.Provider, error) {
	return f.Provider, nil
}

// GetProviderWithOptions implements interface
func (f *Factory) GetProviderWithOptions(useProviderName, key string, relogin bool, loader config.Loader, kubeLoader kubeconfig.Loader, log log.Logger) (cloud.Provider, error) {
	return f.Provider, nil
}

// NewSpaceResumer implements interface
func (f *Factory) NewSpaceResumer(kubeClient kubectl.Client, log log.Logger) resume.SpaceResumer {
	return f.Resumer
}
