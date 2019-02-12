package cloud

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	v1 "github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/util/kubeconfig"
	"github.com/covexo/devspace/pkg/util/log"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// SpaceNameValidationRegEx is the sapace name validation regex
var SpaceNameValidationRegEx = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9-]{1,30}[a-zA-Z0-9]$")

// GetCurrentProvider returns the current specified cloud provider
func GetCurrentProvider(log log.Logger) (*Provider, error) {
	dsConfig := configutil.GetConfig()

	// Don't update or configure anything if we don't use a cloud provider
	if dsConfig.Cluster == nil || dsConfig.Cluster.CloudProvider == nil || *dsConfig.Cluster.CloudProvider == "" {
		return nil, nil
	}

	log.StartWait("Logging into cloud provider...")
	defer log.StopWait()

	// Get provider configuration
	providerConfig, err := ParseCloudConfig()
	if err != nil {
		return nil, err
	}

	// Ensure user is logged in
	err = EnsureLoggedIn(providerConfig, *dsConfig.Cluster.CloudProvider, log)
	if err != nil {
		return nil, err
	}

	// Get provider config
	provider := providerConfig[*dsConfig.Cluster.CloudProvider]

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

	// Get current active config
	activeConfig := generatedConfig.GetActive()

	// Check if there is a space configured
	if activeConfig.SpaceID == nil || *activeConfig.SpaceID == "" {
		return fmt.Errorf("No space for current config %s exists.\n Please run `devspace use space [NAME]` to use an existing space for the configuration or run `devspace create space [NAME]` to create a new space", generatedConfig.ActiveConfig)
	}

	// Update space configuration
	splitted := strings.Split(*activeConfig.SpaceID, ":")
	if len(splitted) != 2 {
		return fmt.Errorf("Malformed space id: %s. Please delete .devspace/generated.yaml and retry", *activeConfig.SpaceID)
	}
	if splitted[0] != provider.Name {
		return fmt.Errorf("Provider name for current config %s does not match between config (Provider name: %s) and configured space (Provider name: %s).\n Please run `devspace use space [NAME] --provider=%s` to use an existing space for the configuration or run `devspace create space [NAME] --provider=%s` to create a new space", splitted[0], *dsConfig.Cluster.CloudProvider, splitted[0], *dsConfig.Cluster.CloudProvider, *dsConfig.Cluster.CloudProvider)
	}

	// Convert id from string to int
	spaceID, err := strconv.Atoi(splitted[1])
	if err != nil {
		return err
	}

	spaceConfig, err := provider.GetSpace(spaceID)
	if err != nil {
		if _, ok := generatedConfig.Spaces[*activeConfig.SpaceID]; ok == false {
			return fmt.Errorf("Couldn't get space config for space id %d: %v", spaceID, err)
		}

		spaceConfig = generatedConfig.Spaces[*activeConfig.SpaceID]
		log.Warnf("Couldn't get space %s: %v", spaceConfig.Name, err)
	} else {
		generatedConfig.Spaces[*activeConfig.SpaceID] = spaceConfig
	}

	return updateDevSpaceConfig(dsConfig, spaceConfig)
}

// ConfigureWithSpaceName configures the environment with the given space name
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

	return updateDevSpaceConfig(dsConfig, spaceConfig)
}

func updateDevSpaceConfig(dsConfig *v1.Config, spaceConfig *generated.SpaceConfig) error {
	// Check if we should use the kubecontext by checking if an api server is specified in the config
	useKubeContext := dsConfig.Cluster == nil || dsConfig.Cluster.CloudProvider == nil || dsConfig.Cluster.APIServer == nil

	// Exchange cluster information
	if useKubeContext {
		kubeContext := DevSpaceKubeContextName + "-" + spaceConfig.Namespace
		dsConfig.Cluster = &v1.Cluster{
			CloudProvider: dsConfig.Cluster.CloudProvider,
		}

		dsConfig.Cluster.Namespace = &spaceConfig.Namespace
		dsConfig.Cluster.KubeContext = &kubeContext

		err := updateKubeConfig(kubeContext, spaceConfig)
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

func updateKubeConfig(contextName string, spaceConfig *generated.SpaceConfig) error {
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

	return kubeconfig.WriteKubeConfig(config, clientcmd.RecommendedHomeFile)
}
