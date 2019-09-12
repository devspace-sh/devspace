package docker

import (
	"context"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/pkg/errors"
)

// NewClient retrieves a new docker client
func NewClient(log log.Logger) (client.CommonAPIClient, error) {
	return NewClientWithMinikube("", false, log)
}

// NewClientWithMinikube creates a new docker client with optionally from the minikube vm
func NewClientWithMinikube(currentKubeContext string, preferMinikube bool, log log.Logger) (client.CommonAPIClient, error) {
	var cli client.CommonAPIClient
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

func newDockerClient() (client.CommonAPIClient, error) {
	cli, err := client.NewClientWithOpts()
	if err != nil {
		return nil, errors.Errorf("Couldn't create docker client: %s", err)
	}

	return cli, nil
}

func newDockerClientFromEnvironment() (client.CommonAPIClient, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, errors.Errorf("Couldn't create docker client: %s", err)
	}

	return cli, nil
}

func newDockerClientFromMinikube(currentKubeContext string) (client.CommonAPIClient, error) {
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

	return client.NewClient(host, env["DOCKER_API_VERSION"], httpclient, nil)
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
