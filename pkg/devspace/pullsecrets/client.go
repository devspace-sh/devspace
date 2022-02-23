package pullsecrets

import (
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
)

// Client communicates with a registry
type Client interface {
	EnsurePullSecrets(ctx *devspacecontext.Context, namespace string) error
	EnsurePullSecret(ctx *devspacecontext.Context, namespace, registryURL string) error

	CreatePullSecret(ctx *devspacecontext.Context, options *PullSecretOptions) error
}

// NewClient creates a client for a registry
func NewClient(dockerClient docker.Client) Client {
	return &client{
		dockerClient: dockerClient,
	}
}

type client struct {
	dockerClient docker.Client
}
