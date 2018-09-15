package builder

import "github.com/docker/docker/api/types"

type BuilderInterface interface {
	Authenticate(username, password string, checkCredentialsStore bool) (*types.AuthConfig, error)
	BuildImage(contextPath, dockerfilePath string, options *types.ImageBuildOptions) error
	PushImage() error
}
