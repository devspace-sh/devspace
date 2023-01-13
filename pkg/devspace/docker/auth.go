package docker

import (
	"context"
	"strings"

	cliTypes "github.com/docker/cli/cli/config/types"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/registry"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	"github.com/pkg/errors"
)

// GetRegistryEndpoint retrieves the correct registry url
func (c *client) GetRegistryEndpoint(ctx context.Context, registryURL string) (bool, string, error) {
	authServer := c.getOfficialServer(ctx)
	if registryURL == "" || registryURL == "hub.docker.com" {
		registryURL = authServer
	}

	return registryURL == authServer, registryURL, nil
}

// GetAuthConfig returns the AuthConfig for a Docker registry from the Docker credential helper
func (c *client) GetAuthConfig(ctx context.Context, registryURL string, checkCredentialsStore bool) (*types.AuthConfig, error) {
	isDefaultRegistry, serverAddress, err := c.GetRegistryEndpoint(ctx, registryURL)
	if err != nil {
		return nil, err
	}

	return getDefaultAuthConfig(checkCredentialsStore, serverAddress, isDefaultRegistry)
}

// Login logs the user into docker
func (c *client) Login(ctx context.Context, registryURL, user, password string, checkCredentialsStore, saveAuthConfig, relogin bool) (*types.AuthConfig, error) {
	isDefaultRegistry, serverAddress, err := c.GetRegistryEndpoint(ctx, registryURL)
	if err != nil {
		return nil, err
	}

	authConfig, err := getDefaultAuthConfig(checkCredentialsStore, serverAddress, isDefaultRegistry)
	if err != nil || authConfig.Username == "" || authConfig.Password == "" || relogin {
		authConfig.Username = strings.TrimSpace(user)
		authConfig.Password = strings.TrimSpace(password)
	}

	// Check if docker is installed
	_, err = c.Ping(ctx)
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
		response, err := c.RegistryLogin(ctx, *authConfig)
		if err != nil {
			return nil, err
		}

		if response.IdentityToken != "" {
			authConfig.Password = ""
			authConfig.IdentityToken = response.IdentityToken
		}
	}

	if saveAuthConfig {
		configfile, err := LoadDockerConfig()
		if err != nil {
			return nil, err
		}

		// convert
		authconfigConverted := &cliTypes.AuthConfig{}
		err = util.Convert(authConfig, authconfigConverted)
		if err != nil {
			return nil, err
		}

		err = configfile.GetCredentialsStore(serverAddress).Store(*authconfigConverted)
		if err != nil {
			return nil, errors.Errorf("Error saving auth info in credentials store: %v", err)
		}

		err = configfile.Save()
		if err != nil {
			return nil, errors.Errorf("Error saving docker config: %v", err)
		}
	}

	return authConfig, nil
}

func (c *client) getOfficialServer(ctx context.Context) string {
	// The daemon `/info` endpoint informs us of the default registry being
	// used. This is essential in cross-platforms environment, where for
	// example a Linux client might be interacting with a Windows daemon, hence
	// the default registry URL might be Windows specific.
	serverAddress := registry.IndexServer
	if info, err := c.Info(ctx); err != nil {
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
		configfile, err := LoadDockerConfig()
		if configfile != nil && err == nil {
			authconfigOrig, err := configfile.GetAuthConfig(serverAddress)
			if err != nil {
				authconfig.ServerAddress = serverAddress
				return &authconfig, err
			}

			// convert
			err = util.Convert(authconfigOrig, &authconfig)
			if err != nil {
				authconfig.ServerAddress = serverAddress
				return &authconfig, err
			}
		}
	}

	authconfig.ServerAddress = serverAddress
	return &authconfig, err
}
