package cloud

import (
	"encoding/base64"
	"regexp"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/stdinutil"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// SpaceNameValidationRegEx is the sapace name validation regex
var SpaceNameValidationRegEx = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9-]{1,30}[a-zA-Z0-9]$")

// GetProvider returns the current specified cloud provider
func GetProvider(useProviderName *string, log log.Logger) (*Provider, error) {
	// Get provider configuration
	providerConfig, err := ParseCloudConfig()
	if err != nil {
		return nil, err
	}

	providerName := DevSpaceCloudProviderName
	if useProviderName == nil {
		// Choose cloud provider
		if len(providerConfig) > 1 {
			options := []string{}
			for providerHost := range providerConfig {
				options = append(options, providerHost)
			}

			providerName = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question: "Select cloud provider",
				Options:  options,
			})
		}
	} else {
		providerName = *useProviderName
	}

	log.StartWait("Logging into cloud provider...")
	defer log.StopWait()

	// Ensure user is logged in
	err = EnsureLoggedIn(providerConfig, providerName, log)
	if err != nil {
		return nil, err
	}

	// Return provider config
	return providerConfig[providerName], nil
}

// GetKubeContextNameFromSpace returns the kube context name for a space
func GetKubeContextNameFromSpace(spaceName string, providerName string) string {
	prefix := DevSpaceKubeContextName
	if providerName != DevSpaceCloudProviderName {
		prefix += "-" + strings.ToLower(strings.Replace(providerName, ".", "-", -1))
	}

	return prefix + "-" + strings.ToLower(spaceName)
}

// UpdateKubeConfig updates the kube config and adds the spaceConfig context
func UpdateKubeConfig(contextName string, spaceConfig *Space, setActive bool) error {
	config, err := kubeconfig.ReadKubeConfig(clientcmd.RecommendedHomeFile)
	if err != nil {
		return err
	}
	caCert, err := base64.StdEncoding.DecodeString(spaceConfig.CaCert)
	if err != nil {
		return err
	}

	cluster := api.NewCluster()
	cluster.Server = spaceConfig.Server
	cluster.CertificateAuthorityData = caCert

	authInfo := api.NewAuthInfo()
	authInfo.Token = spaceConfig.ServiceAccountToken

	config.Clusters[contextName] = cluster
	config.AuthInfos[contextName] = authInfo

	// Update kube context
	context := api.NewContext()
	context.Cluster = contextName
	context.AuthInfo = contextName
	context.Namespace = spaceConfig.Namespace

	config.Contexts[contextName] = context

	if setActive {
		config.CurrentContext = contextName
	}

	return kubeconfig.WriteKubeConfig(config, clientcmd.RecommendedHomeFile)
}
