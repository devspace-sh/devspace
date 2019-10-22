package docker

import (
	"context"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/util/log"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/pkg/errors"
)

//ClientInterface contains all functions required to interact with docker
type ClientInterface interface {
	Ping(ctx context.Context) (dockertypes.Ping, error)
	NegotiateAPIVersion(ctx context.Context)
	ImageBuild(ctx context.Context, context io.Reader, options dockertypes.ImageBuildOptions) (dockertypes.ImageBuildResponse, error)
	ImagePush(ctx context.Context, ref string, options dockertypes.ImagePushOptions) (io.ReadCloser, error)

	Login(registryURL, user, password string, checkCredentialsStore, saveAuthConfig, relogin bool) (*dockertypes.AuthConfig, error)
	DeleteImageByName(imageName string, log log.Logger) ([]dockertypes.ImageDeleteResponseItem, error)
	DeleteImageByFilter(filter filters.Args, log log.Logger) ([]dockertypes.ImageDeleteResponseItem, error)
	GetAuthConfig(registryURL string, checkCredentialsStore bool) (*dockertypes.AuthConfig, error)
}

//Client is a client for docker
type Client struct {
	client.Client
}

// NewClient retrieves a new docker client
func NewClient(log log.Logger) (ClientInterface, error) {
	return NewClientWithMinikube("", false, log)
}

// NewClientWithMinikube creates a new docker client with optionally from the minikube vm
func NewClientWithMinikube(currentKubeContext string, preferMinikube bool, log log.Logger) (ClientInterface, error) {
	if fakeClient != nil {
		return fakeClient, nil
	}
	
	var cli ClientInterface
	var err error

	if preferMinikube {
		cli, err = newDockerClientFromMinikube(currentKubeContext)
	}
	if preferMinikube == false || err != nil {
		cli, err = newDockerClientFromEnvironment()
		if err != nil {
			log.Warnf("Error creating docker client from environment: %v", err)

			// Last try to create it without the environment option
			cli, err = newDockerClient()
			if err != nil {
				return nil, errors.Errorf("Cannot create docker client: %v", err)
			}
		}
	}

	cli.NegotiateAPIVersion(context.Background())
	return cli, nil
}

func newDockerClient() (ClientInterface, error) {
	cli, err := client.NewClientWithOpts()
	if err != nil {
		return nil, errors.Errorf("Couldn't create docker client: %s", err)
	}

	return &Client{*cli}, nil
}

func newDockerClientFromEnvironment() (ClientInterface, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, errors.Errorf("Couldn't create docker client: %s", err)
	}

	return &Client{*cli}, nil
}

func newDockerClientFromMinikube(currentKubeContext string) (ClientInterface, error) {
	if currentKubeContext != "minikube" {
		return nil, errors.New("Cluster is not a minikube cluster")
	}

	env, err := getMinikubeEnvironment()
	if err != nil {
		return nil, err
	}

	var httpclient *http.Client
	if dockerCertPath := env["DOCKER_CERT_PATH"]; dockerCertPath != "" {
		options := tlsconfig.Options{
			CAFile:             filepath.Join(dockerCertPath, "ca.pem"),
			CertFile:           filepath.Join(dockerCertPath, "cert.pem"),
			KeyFile:            filepath.Join(dockerCertPath, "key.pem"),
			InsecureSkipVerify: env["DOCKER_TLS_VERIFY"] == "",
		}
		tlsc, err := tlsconfig.Client(options)
		if err != nil {
			return nil, err
		}

		httpclient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsc,
			},
			CheckRedirect: client.CheckRedirect,
		}
	}

	host := env["DOCKER_HOST"]
	if host == "" {
		host = client.DefaultDockerHost
	}

	cli, err := client.NewClientWithOpts(client.WithHost(host), client.WithVersion(env["DOCKER_API_VERSION"]), client.WithHTTPClient(httpclient), client.WithHTTPHeaders(nil))
	if err != nil {
		return nil, err
	}

	return &Client{*cli}, nil
}

func getMinikubeEnvironment() (map[string]string, error) {
	cmd := exec.Command("minikube", "docker-env", "--shell", "none")
	out, err := cmd.Output()

	if err != nil {
		return nil, err
	}

	env := map[string]string{}

	for _, line := range strings.Split(string(out), "\n") {
		envKeyValue := strings.Split(line, "=")

		if len(envKeyValue) != 2 {
			continue
		}

		env[envKeyValue[0]] = envKeyValue[1]
	}

	return env, nil
}
