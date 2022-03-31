package configure

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/generator"
	"github.com/loft-sh/devspace/pkg/util/kubeconfig"
	"github.com/loft-sh/devspace/pkg/util/log"
)

// Manager controls the devspace configuration
type Manager interface {
	AddKubectlDeployment(deploymentName string, isKustomization bool) error
	AddHelmDeployment(deploymentName string) error
	AddComponentDeployment(deploymentName, image string, servicePort int) error
	AddImage(imageName, image, projectNamespace, dockerfile string, languageHandler *generator.LanguageHandler) error
}

// Factory defines the factory methods needed by the configure manager to create new configuration
type Factory interface {
	NewDockerClientWithMinikube(currentKubeContext string, preferMinikube bool, log log.Logger) (docker.Client, error)
	NewKubeConfigLoader() kubeconfig.Loader
}

type manager struct {
	log        log.Logger
	config     *latest.Config
	localCache localcache.Cache
	factory    Factory
}

// NewManager creates a new instance of the interface Manager
func NewManager(factory Factory, config *latest.Config, localCache localcache.Cache, log log.Logger) Manager {
	return &manager{
		log:        log,
		factory:    factory,
		config:     config,
		localCache: localCache,
	}
}
