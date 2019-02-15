package cloud

import (
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	v1 "github.com/covexo/devspace/pkg/devspace/config/versions/latest"
	"github.com/covexo/devspace/pkg/util/kubeconfig"
	"github.com/covexo/devspace/pkg/util/log"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// SpaceNameValidationRegEx is the sapace name validation regex
var SpaceNameValidationRegEx = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9-]{1,30}[a-zA-Z0-9]$")

// GetCurrentProvider returns the current specified cloud provider
func GetCurrentProvider(log log.Logger) (*Provider, error) {
	log.StartWait("Logging into cloud provider...")
	defer log.StopWait()

	// Get provider configuration
	providerConfig, err := ParseCloudConfig()
	if err != nil {
		return nil, err
	}

	providerName := DevSpaceCloudProviderName
	if configutil.ConfigExists() {
		dsConfig := configutil.GetConfig()

		// Don't update or configure anything if we don't use a cloud provider
		if dsConfig.Cluster == nil || dsConfig.Cluster.CloudProvider == nil || *dsConfig.Cluster.CloudProvider == "" {
			return nil, nil
		}

		providerName = *dsConfig.Cluster.CloudProvider
	}

	// Ensure user is logged in
	err = EnsureLoggedIn(providerConfig, providerName, log)
	if err != nil {
		return nil, err
	}

	// Get provider config
	provider := providerConfig[providerName]

	return provider, nil
}

// Configure will alter the cluster configuration in the generated config
func Configure(log log.Logger) error {
	dsConfig := configutil.GetConfig()

	// Get provider and login
	provider, err := GetCurrentProvider(log)
	if err != nil {
		return err
	}
	if provider == nil {
		return nil
	}

	log.StartWait("Retrieving cloud context...")
	defer log.StopWait()

	// Get generated config
	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		return err
	}

	// Save generated config later
	defer generated.SaveConfig(generatedConfig)

	// Check if there is a space configured
	if generatedConfig.Space == nil {
		return errors.New("No space configured.\n Please run `devspace use space [NAME]` to use an existing space or run `devspace create space [NAME]` to create a new space")
	}

	// Refresh space configuration
	spaceConfig, err := provider.GetSpace(generatedConfig.Space.SpaceID)
	if err != nil {
		spaceConfig = generatedConfig.Space
		log.Warnf("Couldn't get space %s: %v", spaceConfig.Name, err)
	} else {
		generatedConfig.Space = spaceConfig
	}

	return updateDevSpaceConfig(dsConfig, spaceConfig, log)
}

// ConfigureWithSpaceName configures the environment temporarily with the given space name
func ConfigureWithSpaceName(spaceName string, log log.Logger) error {
	dsConfig := configutil.GetConfig()

	// Get provider and login
	provider, err := GetCurrentProvider(log)
	if err != nil {
		return err
	}
	if provider == nil {
		return nil
	}

	log.StartWait("Retrieving cloud context...")
	defer log.StopWait()

	spaceConfig, err := provider.GetSpaceByName(spaceName)
	if err != nil {
		return fmt.Errorf("Couldn't get space config for space %s: %v", spaceName, err)
	}

	return updateDevSpaceConfig(dsConfig, spaceConfig, log)
}

func updateDevSpaceConfig(dsConfig *v1.Config, spaceConfig *generated.SpaceConfig, log log.Logger) error {
	log.Infof("Using space %s", spaceConfig.Name)

	// Check if we should use the kubecontext by checking if an api server is specified in the config
	useKubeContext := dsConfig.Cluster == nil || dsConfig.Cluster.CloudProvider == nil || dsConfig.Cluster.APIServer == nil

	// Exchange cluster information
	if useKubeContext {
		kubeContext := GetKubeContextNameFromSpace(spaceConfig)
		dsConfig.Cluster = &v1.Cluster{
			CloudProvider: dsConfig.Cluster.CloudProvider,
		}

		dsConfig.Cluster.Namespace = &spaceConfig.Namespace
		dsConfig.Cluster.KubeContext = &kubeContext

		err := UpdateKubeConfig(kubeContext, spaceConfig, false)
		if err != nil {
			return err
		}
	} else {
		dsConfig.Cluster = &v1.Cluster{
			CloudProvider: dsConfig.Cluster.CloudProvider,
		}

		dsConfig.Cluster.APIServer = &spaceConfig.Server
		dsConfig.Cluster.Namespace = &spaceConfig.Namespace
		dsConfig.Cluster.CaCert = &spaceConfig.CaCert
		dsConfig.Cluster.User = &v1.ClusterUser{
			Token: &spaceConfig.ServiceAccountToken,
		}
	}

	return nil
}

// GetKubeContextNameFromSpace returns the kube context name for a space
func GetKubeContextNameFromSpace(spaceConfig *generated.SpaceConfig) string {
	return DevSpaceKubeContextName + "-" + strings.ToLower(spaceConfig.Name)
}

// UpdateKubeConfig updates the kube config and adds the spaceConfig context
func UpdateKubeConfig(contextName string, spaceConfig *generated.SpaceConfig, setActive bool) error {
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
