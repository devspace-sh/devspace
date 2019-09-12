package cloud

import (
	"encoding/base64"
	"regexp"
	"strconv"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"k8s.io/client-go/tools/clientcmd/api"
)

// SpaceNameValidationRegEx is the sapace name validation regex
var SpaceNameValidationRegEx = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9-]{1,30}[a-zA-Z0-9]$")

// GetDefaultProviderName returns the default provider name
func GetDefaultProviderName() (string, error) {
	// Get provider configuration
	providerConfig, err := config.ParseProviderConfig()
	if err != nil {
		return "", err
	}

	// Choose cloud provider
	providerName := config.DevSpaceCloudProviderName
	if providerConfig.Default != "" {
		providerName = providerConfig.Default
	}

	return providerName, nil
}

// GetProvider returns the current specified cloud provider
func GetProvider(useProviderName *string, log log.Logger) (*Provider, error) {
	// Get provider configuration
	providerConfig, err := config.ParseProviderConfig()
	if err != nil {
		return nil, err
	}

	providerName := config.DevSpaceCloudProviderName
	if useProviderName == nil {
		// Choose cloud provider
		if providerConfig.Default != "" {
			providerName = providerConfig.Default
		} else if len(providerConfig.Providers) > 1 {
			options := []string{}
			for _, providerHost := range providerConfig.Providers {
				options = append(options, providerHost.Name)
			}

			providerName, err = survey.Question(&survey.QuestionOptions{
				Question: "Select cloud provider",
				Options:  options,
			}, log)
			if err != nil {
				return nil, err
			}
		}
	} else {
		providerName = *useProviderName
	}

	// Ensure user is logged in
	err = EnsureLoggedIn(providerConfig, providerName, log)
	if err != nil {
		return nil, err
	}

	// Set cluster key map
	provider := config.GetProvider(providerConfig, providerName)
	if provider.ClusterKey == nil {
		provider.ClusterKey = make(map[int]string)
	}

	// Return provider config
	return &Provider{*provider}, nil
}

// GetKubeContextNameFromSpace returns the kube context name for a space
func GetKubeContextNameFromSpace(spaceName string, providerName string) string {
	prefix := DevSpaceKubeContextName
	if providerName != config.DevSpaceCloudProviderName {
		prefix += "-" + strings.ToLower(strings.Replace(providerName, ".", "-", -1))
	}

	// Replace : with - for usernames
	spaceName = strings.Replace(spaceName, ":", "-", -1)
	return prefix + "-" + strings.ToLower(spaceName)
}

// UpdateKubeConfig updates the kube config and adds the spaceConfig context
func UpdateKubeConfig(contextName string, serviceAccount *latest.ServiceAccount, spaceID int, providerName string, setActive bool) error {
	config, err := kubeconfig.LoadRawConfig()
	if err != nil {
		return err
	}
	caCert, err := base64.StdEncoding.DecodeString(serviceAccount.CaCert)
	if err != nil {
		return err
	}

	cluster := api.NewCluster()
	cluster.Server = serviceAccount.Server
	cluster.CertificateAuthorityData = caCert

	authInfo := api.NewAuthInfo()
	authInfo.Exec = &api.ExecConfig{
		APIVersion: "client.authentication.k8s.io/v1alpha1",
		Command:    kubeconfig.AuthCommand,
		Args:       []string{"use", "space", "--provider", providerName, "--space-id", strconv.Itoa(spaceID), "--get-token", "--silent"},
	}

	config.Clusters[contextName] = cluster
	config.AuthInfos[contextName] = authInfo

	// Update kube context
	context := api.NewContext()
	context.Cluster = contextName
	context.AuthInfo = contextName
	context.Namespace = serviceAccount.Namespace

	config.Contexts[contextName] = context

	if setActive {
		config.CurrentContext = contextName
	}

	return kubeconfig.SaveConfig(config)
}
