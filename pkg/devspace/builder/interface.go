package builder

import "github.com/docker/docker/api/types"

// Interface defines methods for builders (e.g. docker, kaniko)
type Interface interface {
	Authenticate(username, password string, checkCredentialsStore bool) (*types.AuthConfig, error)
	BuildImage(contextPath, dockerfilePath string, options *types.ImageBuildOptions) error
	PushImage() error
}
