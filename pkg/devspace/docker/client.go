package docker

import (
	"context"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devspace/pkg/util/log"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/pkg/errors"
)

var errNotMinikube = errors.New("not a minikube context")

// Client contains all functions required to interact with docker
type Client interface {
	Ping(ctx context.Context) (dockertypes.Ping, error)
	NegotiateAPIVersion(ctx context.Context)

	ImageBuild(ctx context.Context, context io.Reader, options dockertypes.ImageBuildOptions) (dockertypes.ImageBuildResponse, error)
	ImageBuildCLI(useBuildkit bool, context io.Reader, writer io.Writer, additionalArgs []string, options dockertypes.ImageBuildOptions, log log.Logger) error

	ImagePush(ctx context.Context, ref string, options dockertypes.ImagePushOptions) (io.ReadCloser, error)

	Login(registryURL, user, password string, checkCredentialsStore, saveAuthConfig, relogin bool) (*dockertypes.AuthConfig, error)
	GetAuthConfig(registryURL string, checkCredentialsStore bool) (*dockertypes.AuthConfig, error)

	ParseProxyConfig(buildArgs map[string]*string) map[string]*string

	DeleteImageByName(imageName string, log log.Logger) ([]dockertypes.ImageDeleteResponseItem, error)
	DeleteImageByFilter(filter filters.Args, log log.Logger) ([]dockertypes.ImageDeleteResponseItem, error)
}

//Client is a client for docker
type client struct {
	dockerclient.CommonAPIClient
}

// NewClient retrieves a new docker client
func NewClient(log log.Logger) (Client, error) {
	return NewClientWithMinikube("", false, log)
}

// NewClientWithMinikube creates a new docker client with optionally from the minikube vm
func NewClientWithMinikube(currentKubeContext string, preferMinikube bool, log log.Logger) (Client, error) {
	var cli Client
	var err error

	if preferMinikube {
		cli, err = newDockerClientFromMinikube(currentKubeContext)
		if err != nil && err != errNotMinikube {
			log.Warnf("Error creating minikube docker client: %v", err)
		}
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

func newDockerClient() (Client, error) {
	cli, err := dockerclient.NewClientWithOpts()
	if err != nil {
		return nil, errors.Errorf("Couldn't create docker client: %s", err)
	}

	return &client{cli}, nil
}

func newDockerClientFromEnvironment() (Client, error) {
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv)
	if err != nil {
		return nil, errors.Errorf("Couldn't create docker client: %s", err)
	}

	return &client{cli}, nil
}

func newDockerClientFromMinikube(currentKubeContext string) (Client, error) {
	if currentKubeContext != "minikube" {
		return nil, errNotMinikube
	}

	env, err := getMinikubeEnvironment()
	if err != nil {
		return nil, errors.Errorf("can't retrieve minikube docker environment due to error: %v", err)
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
			CheckRedirect: dockerclient.CheckRedirect,
		}
	}

	host := env["DOCKER_HOST"]
	if host == "" {
		host = dockerclient.DefaultDockerHost
	}

	cli, err := dockerclient.NewClientWithOpts(dockerclient.WithHost(host), dockerclient.WithVersion(env["DOCKER_API_VERSION"]), dockerclient.WithHTTPClient(httpclient), dockerclient.WithHTTPHeaders(nil))
	if err != nil {
		return nil, err
	}

	return &client{cli}, nil
}

func getMinikubeEnvironment() (map[string]string, error) {
	cmd := exec.Command("minikube", "docker-env", "--shell", "none")
	out, err := cmd.Output()

	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			out = ee.Stderr
		}
		return nil, errors.Errorf("error executing 'minikube docker-env --shell none'\nerror: %v\noutput: %s", err, string(out))
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

// ParseProxyConfig parses the proxy config from the ~/.docker/config.json
func (c *client) ParseProxyConfig(buildArgs map[string]*string) map[string]*string {
	dockerConfig, err := loadDockerConfig()
	if err == nil {
		buildArgs = dockerConfig.ParseProxyConfig(c.DaemonHost(), buildArgs)
	}

	return buildArgs
}
