package kubeconfig

import (
	"encoding/base64"
	"fmt"
	"strconv"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// Name of the command used to get auth token for kube-context of Spaces
const AuthCommand = "devspace"

// ConfigExists checks if a kube config exists
func ConfigExists() bool {
	return clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename() != ""
}

// LoadConfig loads the kube config with the default loading rules
func LoadConfig() clientcmd.ClientConfig {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
}

// LoadConfigFromContext loads the kube client config from a certain context
func LoadConfigFromContext(context string) (clientcmd.ClientConfig, error) {
	kubeConfig, err := LoadRawConfig()
	if err != nil {
		return nil, err
	}

	return clientcmd.NewNonInteractiveClientConfig(*kubeConfig, context, &clientcmd.ConfigOverrides{}, clientcmd.NewDefaultClientConfigLoadingRules()), nil
}

// LoadRawConfig loads the raw kube config with the default loading rules
func LoadRawConfig() (*api.Config, error) {
	config, err := LoadConfig().RawConfig()
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveConfig writes the kube config back to the specified filename
func SaveConfig(config *api.Config) error {
	return clientcmd.ModifyConfig(clientcmd.NewDefaultClientConfigLoadingRules(), *config, false)
}

// LoadNewConfig creates a new config from scratch with the given parameters and loads it
func LoadNewConfig(contextName, server, caCert, token, namespace string) (clientcmd.ClientConfig, error) {
	config := api.NewConfig()
	decodedCaCert, err := base64.StdEncoding.DecodeString(caCert)
	if err != nil {
		return nil, err
	}

	cluster := api.NewCluster()
	cluster.Server = server
	cluster.CertificateAuthorityData = decodedCaCert

	authInfo := api.NewAuthInfo()
	authInfo.Token = token

	config.Clusters[contextName] = cluster
	config.AuthInfos[contextName] = authInfo

	// Update kube context
	context := api.NewContext()
	context.Cluster = contextName
	context.AuthInfo = contextName

	if namespace != "" {
		context.Namespace = namespace
	}

	config.Contexts[contextName] = context
	config.CurrentContext = contextName

	return clientcmd.NewNonInteractiveClientConfig(*config, contextName, &clientcmd.ConfigOverrides{}, clientcmd.NewDefaultClientConfigLoadingRules()), nil
}

// IsCloudSpace returns true of this context belongs to a Space created by DevSpace Cloud
func IsCloudSpace(context *api.Context) (bool, error) {
	// Get AuthInfo for context
	authInfo, err := GetAuthInfo(context)
	if err != nil {
		return false, fmt.Errorf("Unable to get AuthInfo for kube-context: %v", err)
	}

	if authInfo.Exec.Command == AuthCommand {
		return true, nil
	}
	return false, nil
}

// GetSpaceID returns the id of the Space that belongs to the context with this name
func GetSpaceID(context *api.Context) (int, string, error) {
	// Get AuthInfo for context
	authInfo, err := GetAuthInfo(context)
	if err != nil {
		return 0, "", fmt.Errorf("Unable to get AuthInfo for kube-context: %v", err)
	}

	if authInfo.Exec.Command != AuthCommand {
		return 0, "", fmt.Errorf("Kube-context does not belong to a Space")
	}

	if len(authInfo.Exec.Args) < 6 {
		return 0, "", fmt.Errorf("Kube-context is misconfigured. Please run `devspace use space [SPACE_NAME]` to setup a new context")
	}
	spaceID, err := strconv.Atoi(authInfo.Exec.Args[5])

	return spaceID, authInfo.Exec.Args[3], err
}

// GetCurrentContextSpaceID returns the id and the provider of the Space that belongs to the current context
func GetCurrentContextSpaceID() (int, string, error) {
	currentContext, _, err := GetCurrentContext()
	if err != nil {
		return 0, "", fmt.Errorf("Unable to get current kube-context: %v", err)
	}

	return GetSpaceID(currentContext)
}

// GetAuthInfo returns the AuthInfo of the context with this name
func GetAuthInfo(context *api.Context) (*api.AuthInfo, error) {
	// Load kube-config
	kubeConfig, err := LoadRawConfig()
	if err != nil {
		return nil, err
	}

	// Get AuthInfo for context
	authInfo, ok := kubeConfig.AuthInfos[context.AuthInfo]
	if !ok {
		return nil, fmt.Errorf("Unable to find user information for context in kube-config file")
	}
	return authInfo, nil
}

// GetCurrentContext returns the current kube-context
func GetCurrentContext() (*api.Context, string, error) {
	return GetContext("")
}

// GetContext returns the kube-context with the provided name
func GetContext(name string) (*api.Context, string, error) {
	// Load kube-config
	kubeConfig, err := LoadRawConfig()
	if err != nil {
		return nil, "", err
	}

	// use current context name if none is provided
	if name == "" {
		name = kubeConfig.CurrentContext
	}

	// Get context
	context, ok := kubeConfig.Contexts[name]
	if !ok {
		return nil, "", fmt.Errorf("Unable to find current kube-context '%s' in kube-config file", name)
	}
	return context, name, nil
}
