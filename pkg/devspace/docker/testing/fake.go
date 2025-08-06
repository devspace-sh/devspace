package docker

import (
	"bytes"
	"context"
	"io"
	"strings"
	
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	dockerregistry "github.com/docker/docker/api/types/registry"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/docker/docker/api/types/build"
)

// FakeClient is a prototype for a fake docker cient for testing purposes
type FakeClient struct {
	AuthConfig *dockerregistry.AuthConfig
	PingErr    error
}

// Ping is a fake implementation
func (client *FakeClient) Ping(ctx context.Context) (dockertypes.Ping, error) {
	return dockertypes.Ping{}, client.PingErr
}

// NegotiateAPIVersion is a fake implementation
func (client *FakeClient) NegotiateAPIVersion(ctx context.Context) {}

// ImageBuildCLI builds an image with the docker cli
func (client *FakeClient) ImageBuildCLI(ctx context.Context, workingDir string, useBuildkit bool, context io.Reader, writer io.Writer, additionalArgs []string, options build.ImageBuildOptions, log log.Logger) error {
	return nil
}

// ParseProxyConfig implements the interface
func (client *FakeClient) ParseProxyConfig(buildArgs map[string]*string) map[string]*string {
	return buildArgs
}

// ImageBuild is a fake implementation
func (client *FakeClient) ImageBuild(ctx context.Context, context io.Reader, options build.ImageBuildOptions) (build.ImageBuildResponse, error) {
	return build.ImageBuildResponse{
		Body: io.NopCloser(bytes.NewBufferString("")),
	}, nil
}

// ImagePush is a fake implementation
func (client *FakeClient) ImagePush(ctx context.Context, ref string, options image.PushOptions) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewBufferString("")), nil
}

// Login is a fake implementation
func (client *FakeClient) Login(registryURL, user, password string, checkCredentialsStore, saveAuthConfig, relogin bool) (*dockerregistry.AuthConfig, error) {
	return client.AuthConfig, nil
}

// DeleteImageByName is a fake implementation
func (client *FakeClient) DeleteImageByName(imageName string, log log.Logger) ([]image.DeleteResponse, error) {
	return client.DeleteImageByFilter(filters.NewArgs(filters.Arg("reference", strings.TrimSpace(imageName))), log)
}

// DeleteImageByFilter is a fake implementation
func (client *FakeClient) DeleteImageByFilter(filter filters.Args, log log.Logger) ([]image.DeleteResponse, error) {
	return []image.DeleteResponse{}, nil
}

// GetAuthConfig is a fake implementation
func (client *FakeClient) GetAuthConfig(registryURL string, checkCredentialsStore bool) (*dockerregistry.AuthConfig, error) {
	return client.AuthConfig, nil
}
