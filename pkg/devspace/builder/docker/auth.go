package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/registry"
)

func getOfficialServer(ctx context.Context, client client.CommonAPIClient) string {
	// The daemon `/info` endpoint informs us of the default registry being
	// used. This is essential in cross-platforms environment, where for
	// example a Linux client might be interacting with a Windows daemon, hence
	// the default registry URL might be Windows specific.
	serverAddress := registry.IndexServer
	if info, err := client.Info(ctx); err != nil {
		// Only report the warning if we're in debug mode to prevent nagging during engine initialization workflows
		fmt.Fprintf(log.GetInstance(), "Warning: failed to get default registry endpoint from daemon (%v). Using system default: %s\n", err, serverAddress)
	} else if info.IndexServerAddress == "" {
		fmt.Fprintf(log.GetInstance(), "Warning: Empty registry endpoint from daemon. Using system default: %s\n", serverAddress)
	} else {
		serverAddress = info.IndexServerAddress
	}

	return serverAddress
}

func encodeAuthToBase64(authConfig types.AuthConfig) (string, error) {
	buf, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}

func getDefaultAuthConfig(client client.CommonAPIClient, checkCredStore bool, serverAddress string, isDefaultRegistry bool) (*types.AuthConfig, error) {
	var authconfig types.AuthConfig
	var err error

	configfile, _ := loadDockerConfig()

	if !isDefaultRegistry {
		serverAddress = registry.ConvertToHostname(serverAddress)
	}

	if checkCredStore {
		authconfig, err = configfile.GetAuthConfig(serverAddress)
	} else {
		authconfig = types.AuthConfig{}
	}

	authconfig.ServerAddress = serverAddress
	authconfig.IdentityToken = ""
	return &authconfig, err
}
