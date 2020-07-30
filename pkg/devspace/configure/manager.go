package configure

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// Manager controls the devspace configuration
type Manager interface {
	NewDockerfileComponentDeployment(generatedConfig *generated.Config, name, imageName, dockerfile, context string) (*latest.ImageConfig, *latest.DeploymentConfig, error)
	NewImageComponentDeployment(name, imageName string) (*latest.ImageConfig, *latest.DeploymentConfig, error)
	NewKubectlDeployment(name, manifests string) (*latest.DeploymentConfig, error)
	NewHelmDeployment(name, chartName, chartRepo, chartVersion string) (*latest.DeploymentConfig, error)
	RemoveDeployment(removeAll bool, name string) (bool, error)

	AddImage(nameInConfig, name, tag, contextPath, dockerfilePath, buildTool string) error
	RemoveImage(removeAll bool, names []string) error

	AddPort(namespace, labelSelector string, args []string) error
	RemovePort(removeAll bool, labelSelector string, args []string) error

	AddSyncPath(localPath, containerPath, namespace, labelSelector, excludedPathsString string) error
	RemoveSyncPath(removeAll bool, localPath, containerPath, labelSelector string) error
}

// Factory defines the factory methods needed by the configure manager to create new configuration
type Factory interface {
	NewDockerClientWithMinikube(currentKubeContext string, preferMinikube bool, log log.Logger) (docker.Client, error)
	NewKubeConfigLoader() kubeconfig.Loader
}

type manager struct {
	log     log.Logger
	config  *latest.Config
	factory Factory
}

// NewManager creates a new instance of the interface Manager
func NewManager(factory Factory, config *latest.Config, log log.Logger) Manager {
	return &manager{
		log:     log,
		factory: factory,
		config:  config,
	}
}
