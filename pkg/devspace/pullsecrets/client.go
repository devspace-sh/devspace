package pullsecrets

import (
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
)

// Client communicates with a registry
type Client interface {
	EnsurePullSecrets(ctx devspacecontext.Context, dockerClient docker.Client, pullSecrets []string) error
	EnsurePullSecret(ctx devspacecontext.Context, dockerClient docker.Client, namespace, registryURL string) error
}

// NewClient creates a client for a registry
func NewClient() Client {
	return &client{}
}

type client struct{}
