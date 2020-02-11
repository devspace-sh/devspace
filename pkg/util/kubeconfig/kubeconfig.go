package kubeconfig

import (
	"strconv"
	"sync"

	"github.com/pkg/errors"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

var loadOnceMutext sync.Mutex
var loadOnce sync.Once
var loadedConfig clientcmd.ClientConfig

// AuthCommand is the name of the command used to get auth token for kube-context of Spaces
const AuthCommand = "devspace"

// NewConfig loads a new kube config
func (l *loader) NewConfig() clientcmd.ClientConfig {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
}

// LoadConfig loads the kube config with the default loading rules
func (l *loader) LoadConfig() clientcmd.ClientConfig {
	loadOnceMutext.Lock()
	defer loadOnceMutext.Unlock()

	loadOnce.Do(func() {
		loadedConfig = l.NewConfig()
	})

	return loadedConfig
}

// LoadRawConfig loads the raw kube config with the default loading rules
func (l *loader) LoadRawConfig() (*api.Config, error) {
	config, err := l.LoadConfig().RawConfig()
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// GetCurrentContext retrieves the current kube context
func (l *loader) GetCurrentContext() (string, error) {
	config, err := l.LoadRawConfig()
	if err != nil {
		return "", err
	}

	return config.CurrentContext, nil
}

// SaveConfig writes the kube config back to the specified filename
func (l *loader) SaveConfig(config *api.Config) error {
	loadOnceMutext.Lock()
	defer loadOnceMutext.Unlock()

	err := clientcmd.ModifyConfig(clientcmd.NewDefaultClientConfigLoadingRules(), *config, false)
	if err != nil {
		return err
	}

	// Since the config has changed now we reset the loadOnce
	loadOnce = sync.Once{}
	return nil
}

// IsCloudSpace returns true of this context belongs to a Space created by DevSpace Cloud
func (l *loader) IsCloudSpace(context string) (bool, error) {
	kubeConfig, err := l.LoadRawConfig()
	if err != nil {
		return false, err
	}

	// Get AuthInfo for context
	authInfo, err := getAuthInfo(kubeConfig, context)
	if err != nil {
		return false, errors.Errorf("Unable to get AuthInfo for kube-context: %v", err)
	}

	return authInfo.Exec != nil && authInfo.Exec.Command == AuthCommand, nil
}

// GetSpaceID returns the id of the Space and the cloud provider URL that belongs to the context with this name
func (l *loader) GetSpaceID(context string) (int, string, error) {
	kubeConfig, err := l.LoadRawConfig()
	if err != nil {
		return 0, "", err
	}

	// Get AuthInfo for context
	authInfo, err := getAuthInfo(kubeConfig, context)
	if err != nil {
		return 0, "", errors.Errorf("Unable to get AuthInfo for kube-context: %v", err)
	}

	if authInfo.Exec == nil || authInfo.Exec.Command != AuthCommand {
		return 0, "", errors.Errorf("Kube-context does not belong to a Space")
	}

	if len(authInfo.Exec.Args) < 6 {
		return 0, "", errors.Errorf("Kube-context is misconfigured. Please run `devspace use space [SPACE_NAME]` to setup a new context")
	}
	spaceID, err := strconv.Atoi(authInfo.Exec.Args[5])

	return spaceID, authInfo.Exec.Args[3], err
}

// getAuthInfo returns the AuthInfo of the context with this name
func getAuthInfo(kubeConfig *api.Config, context string) (*api.AuthInfo, error) {
	// Get context
	contextRaw, ok := kubeConfig.Contexts[context]
	if !ok {
		return nil, errors.Errorf("Unable to find kube-context '%s' in kube-config file", context)
	}

	// Get AuthInfo for context
	authInfo, ok := kubeConfig.AuthInfos[contextRaw.AuthInfo]
	if !ok {
		return nil, errors.Errorf("Unable to find user information for context in kube-config file")
	}

	return authInfo, nil
}

// DeleteKubeContext removes the specified devspace id from the kube context if it exists
func (l *loader) DeleteKubeContext(kubeConfig *api.Config, kubeContext string) error {
	// Get context
	contextRaw, ok := kubeConfig.Contexts[kubeContext]
	if !ok {
		// return errors.Errorf("Unable to find current kube-context '%s' in kube-config file", kubeContext)
		// This is debatable but usually we don't care when the context is not there
		return nil
	}

	// Remove context
	delete(kubeConfig.Contexts, kubeContext)

	removeAuthInfo := true
	removeCluster := true

	// Check if AuthInfo or Cluster is used by any other context
	for name, ctx := range kubeConfig.Contexts {
		if name != kubeContext && ctx.AuthInfo == contextRaw.AuthInfo {
			removeAuthInfo = false
		}

		if name != kubeContext && ctx.Cluster == contextRaw.Cluster {
			removeCluster = false
		}
	}

	// Remove AuthInfo if not used by any other context
	if removeAuthInfo {
		delete(kubeConfig.AuthInfos, contextRaw.AuthInfo)
	}

	// Remove Cluster if not used by any other context
	if removeCluster {
		delete(kubeConfig.Clusters, contextRaw.Cluster)
	}

	if kubeConfig.CurrentContext == kubeContext {
		kubeConfig.CurrentContext = ""

		if len(kubeConfig.Contexts) > 0 {
			for context, contextObj := range kubeConfig.Contexts {
				if contextObj != nil {
					kubeConfig.CurrentContext = context
					break
				}
			}
		}
	}

	return nil
}
