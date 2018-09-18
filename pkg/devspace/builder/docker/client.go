package docker

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/covexo/devspace/pkg/devspace/clients/kubectl"

	"github.com/docker/docker/api"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
)

func newDockerClientFromEnvironment() (client.CommonAPIClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("Error getting docker client: %s", err)
	}

	cli.NegotiateAPIVersion(context.Background())
	return cli, nil
}

func newDockerClientFromMinikube() (client.CommonAPIClient, error) {
	if kubectl.IsMinikube() == false {
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
	version := env["DOCKER_API_VERSION"]
	if version == "" {
		version = api.DefaultVersion
	}

	return client.NewClient(host, version, httpclient, nil)
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
