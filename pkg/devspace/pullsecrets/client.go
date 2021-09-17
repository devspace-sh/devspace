package pullsecrets

import (
	config2 "github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/log"
)

// Client communicates with a registry
type Client interface {
	CreatePullSecrets() error
	CreatePullSecret(options *PullSecretOptions) error
}

// NewClient creates a client for a registry
func NewClient(config config2.Config, dependencies []types.Dependency, kubeClient kubectl.Client, dockerClient docker.Client, log log.Logger) Client {
	return &client{
		config:       config,
		dependencies: dependencies,
		kubeClient:   kubeClient,
		dockerClient: dockerClient,
		log:          log,
	}
}

type client struct {
	config       config2.Config
	dependencies []types.Dependency
	kubeClient   kubectl.Client
	dockerClient docker.Client
	log          log.Logger
}
