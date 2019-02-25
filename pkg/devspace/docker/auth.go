package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/registry"
)

// GetRegistryEndpoint retrieves the correct registry url
func GetRegistryEndpoint(client client.CommonAPIClient, registryURL string) (bool, string, error) {
	authServer := getOfficialServer(context.Background(), client)
	if registryURL == "" || registryURL == "hub.docker.com" {
		registryURL = authServer
	}

	return registryURL == authServer, registryURL, nil
}

// GetAuthConfig returns the AuthConfig for a Docker registry from the Docker credential helper
func GetAuthConfig(client client.CommonAPIClient, registryURL string, checkCredentialsStore bool) (*types.AuthConfig, error) {
	isDefaultRegistry, serverAddress, err := GetRegistryEndpoint(client, registryURL)
	if err != nil {
		return nil, err
	}

	return getDefaultAuthConfig(checkCredentialsStore, serverAddress, isDefaultRegistry)
}

// Login logs the user into docker
func Login(client client.CommonAPIClient, registryURL, user, password string, checkCredentialsStore, saveAuthConfig, relogin bool) (*types.AuthConfig, error) {
	ctx := context.Background()
	isDefaultRegistry, serverAddress, err := GetRegistryEndpoint(client, registryURL)
	if err != nil {
		return nil, err
	}

	authConfig, err := getDefaultAuthConfig(checkCredentialsStore, serverAddress, isDefaultRegistry)
	authConfig.IdentityToken = ""
	if err != nil || authConfig.Username == "" || authConfig.Password == "" || relogin {
		authConfig.Username = strings.TrimSpace(user)
		authConfig.Password = strings.TrimSpace(password)
	}

	// Check if docker is installed
	_, err = client.Ping(ctx)
	if err != nil {
		// Docker is not installed, we cannot use client
		service, err := registry.NewService(registry.ServiceOptions{})
		if err != nil {
			return nil, err
		}

		_, token, err := service.Auth(ctx, authConfig, "")
		if err != nil {
			return nil, err
		}

		if token != "" {
			authConfig.Password = ""
			authConfig.IdentityToken = token
		}
	} else {
		// Docker is installed, we can use client
		response, err := client.RegistryLogin(ctx, *authConfig)
		if err != nil {
			return nil, err
		}

		if response.IdentityToken != "" {
			authConfig.Password = ""
			authConfig.IdentityToken = response.IdentityToken
		}
	}

	if saveAuthConfig {
		configfile, err := loadDockerConfig()
		if err != nil {
			return nil, err
		}

		err = configfile.GetCredentialsStore(serverAddress).Store(*authConfig)
		if err != nil {
			return nil, fmt.Errorf("Error saving auth info in credentials store: %v", err)
		}

		err = configfile.Save()
		if err != nil {
			return nil, fmt.Errorf("Error saving docker config: %v", err)
		}
	}

	return authConfig, nil
}

func getOfficialServer(ctx context.Context, client client.CommonAPIClient) string {
	// The daemon `/info` endpoint informs us of the default registry being
	// used. This is essential in cross-platforms environment, where for
	// example a Linux client might be interacting with a Windows daemon, hence
	// the default registry URL might be Windows specific.
	serverAddress := registry.IndexServer
	if info, err := client.Info(ctx); err != nil {
		// Only report the warning if we're in debug mode to prevent nagging during engine initialization workflows
		// log.Warnf("Warning: failed to get default registry endpoint from daemon (%v). Using system default: %s", err, serverAddress)
	} else if info.IndexServerAddress == "" {
		// log.Warnf("Warning: Empty registry endpoint from daemon. Using system default: %s", serverAddress)
	} else {
		serverAddress = info.IndexServerAddress
	}

	return serverAddress
}

func getDefaultAuthConfig(checkCredStore bool, serverAddress string, isDefaultRegistry bool) (*types.AuthConfig, error) {
	var authconfig types.AuthConfig
	var err error

	if !isDefaultRegistry {
		serverAddress = registry.ConvertToHostname(serverAddress)
	}

	if checkCredStore {
		configfile, err := loadDockerConfig()

		if configfile != nil && err == nil {
			authconfig, err = configfile.GetAuthConfig(serverAddress)
		}
	}

	authconfig.ServerAddress = serverAddress
	return &authconfig, err
}
