package configure

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/util/kubeconfig"
	"github.com/loft-sh/devspace/pkg/util/log"
)

// Manager controls the devspace configuration
type Manager interface {
	NewDockerfileComponentDeployment(name, imageName, dockerfile, context string) (*latest.ImageConfig, *latest.DeploymentConfig, error)
	NewKubectlDeployment(name, manifests string) (*latest.DeploymentConfig, error)
	NewHelmDeployment(name, chartName, chartRepo, chartVersion string) (*latest.DeploymentConfig, error)
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
