package docker

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

var fakeClient Client

// SetFakeClient causes NewClient to return the given fake client. !ONLY FOR TESTING!
func SetFakeClient(fake Client) {
	fakeClient = fake
}

// FakeClient is a prototype for a fake docker cient for testing purposes
type FakeClient struct {
	AuthConfig *dockertypes.AuthConfig
}

// Ping is a fake implementation
func (client *FakeClient) Ping(ctx context.Context) (dockertypes.Ping, error) {
	return dockertypes.Ping{}, nil
}

// NegotiateAPIVersion is a fake implementation
func (client *FakeClient) NegotiateAPIVersion(ctx context.Context) {}

// ImageBuildCLI builds an image with the docker cli
func (client *FakeClient) ImageBuildCLI(useBuildkit bool, context io.Reader, writer io.Writer, options dockertypes.ImageBuildOptions) error {
	return nil
}

// ImageBuild is a fake implementation
func (client *FakeClient) ImageBuild(ctx context.Context, context io.Reader, options dockertypes.ImageBuildOptions) (dockertypes.ImageBuildResponse, error) {
	return dockertypes.ImageBuildResponse{
		Body: ioutil.NopCloser(bytes.NewBufferString("")),
	}, nil
}

// ImagePush is a fake implementation
func (client *FakeClient) ImagePush(ctx context.Context, ref string, options dockertypes.ImagePushOptions) (io.ReadCloser, error) {
	return ioutil.NopCloser(bytes.NewBufferString("")), nil
}

// Login is a fake implementation
func (client *FakeClient) Login(registryURL, user, password string, checkCredentialsStore, saveAuthConfig, relogin bool) (*dockertypes.AuthConfig, error) {
	return client.AuthConfig, nil
}

// DeleteImageByName is a fake implementation
func (client *FakeClient) DeleteImageByName(imageName string, log log.Logger) ([]dockertypes.ImageDeleteResponseItem, error) {
	return client.DeleteImageByFilter(filters.NewArgs(filters.Arg("reference", strings.TrimSpace(imageName))), log)
}

// DeleteImageByFilter is a fake implementation
func (client *FakeClient) DeleteImageByFilter(filter filters.Args, log log.Logger) ([]dockertypes.ImageDeleteResponseItem, error) {
	return []dockertypes.ImageDeleteResponseItem{}, nil
}

// GetAuthConfig is a fake implementation
func (client *FakeClient) GetAuthConfig(registryURL string, checkCredentialsStore bool) (*dockertypes.AuthConfig, error) {
	return client.AuthConfig, nil
}
