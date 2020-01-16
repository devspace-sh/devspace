package configure

import (
	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
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
	NewPredefinedComponentDeployment(name, component string) (*latest.DeploymentConfig, error)
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

type manager struct {
	log               log.Logger
	config            *latest.Config
	kubeLoader        kubeconfig.Loader
	cloudConfigLoader cloudconfig.Loader
	dockerClient      docker.Client
}

// NewManager creates a new instance of the interface Manager
func NewManager(config *latest.Config, log log.Logger) Manager {
	return &manager{
		log:    log,
		config: config,
	}
}
