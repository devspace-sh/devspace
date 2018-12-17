package cloud

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"regexp"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/util/kubeconfig"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// DevSpaceNameValidationRegEx is the devsapace name validation regex
var DevSpaceNameValidationRegEx = regexp.MustCompile("^[a-zA-Z0-9-]{3,32}$")

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

// Configure will alter the cluster configuration in the config
func Configure(useKubeContext, dry bool, log log.Logger) error {
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

	// Check if we have to create devspace
	if generatedConfig.Cloud == nil || generatedConfig.Cloud.ProviderName != *dsConfig.Cluster.CloudProvider {
		if dry {
			return errors.New("No devspace configured")
		}

		dir, err := os.Getwd()
		if err != nil {
			return errors.Wrap(err, "cloud configure")
		}

		devSpaceName := filepath.Base(dir)
		reg := regexp.MustCompile("[^a-zA-Z0-9]+")

		devSpaceName = reg.ReplaceAllString(devSpaceName, "")
		if DevSpaceNameValidationRegEx.MatchString(devSpaceName) == false {
			devSpaceName = "devspace"
		}

		devSpaceID, err := provider.CreateDevSpace(devSpaceName)
		if err != nil {
			return err
		}

		generatedConfig.Cloud = &generated.CloudConfig{
			DevSpaceID:   devSpaceID,
			ProviderName: *dsConfig.Cluster.CloudProvider,
			Name:         devSpaceName,
			Targets:      make(map[string]*generated.DevSpaceTargetConfig),
		}
	}

	// Check if we have to create devspace target
	target := configutil.GetCurrentCloudTarget(dsConfig)
	if target == nil {
		return errors.New("No cloud target specified")
	}

	targetConfig := generatedConfig.Cloud.Targets[*target]
	if targetConfig == nil {
		if dry {
			return errors.New("No devspace target configured")
		}

		// Check if it is there remotely
		_, err := provider.GetDevSpaceTargetConfig(generatedConfig.Cloud.DevSpaceID, *target)
		if err != nil {
			err = provider.CreateDevSpaceTarget(generatedConfig.Cloud.DevSpaceID, *target)
			if err != nil {
				return errors.Wrap(err, "cloud configure")
			}
		}
	}

	newTargetConfig, err := provider.GetDevSpaceTargetConfig(generatedConfig.Cloud.DevSpaceID, *target)
	if err != nil {
		log.Warnf("Couldn't retrieve devspace target config: %v", err)
	}
	if targetConfig == nil && newTargetConfig == nil {
		return errors.New("Couldn't retrieve devspace target config")
	}
	if newTargetConfig != nil {
		generatedConfig.Cloud.Targets[*target] = newTargetConfig
	}

	// Configure devspace config
	err = updateDevSpaceConfig(useKubeContext, dsConfig, generatedConfig.Cloud.Targets[*target])
	if err != nil {
		return err
	}

	return nil
}

func updateDevSpaceConfig(useKubeContext bool, dsConfig *v1.Config, targetConfig *generated.DevSpaceTargetConfig) error {
	// Update tiller if needed
	if dsConfig.Tiller != nil && dsConfig.Tiller.Namespace != nil {
		*dsConfig.Tiller.Namespace = targetConfig.Namespace
	}

	// Update registry namespace if needed
	if dsConfig.InternalRegistry != nil && dsConfig.InternalRegistry.Namespace != nil {
		*dsConfig.InternalRegistry.Namespace = targetConfig.Namespace
	}

	// Exchange cluster information
	if useKubeContext {
		kubeContext := DevSpaceKubeContextName + "-" + targetConfig.Namespace
		dsConfig.Cluster = &v1.Cluster{
			CloudProvider: dsConfig.Cluster.CloudProvider,
			CloudTarget:   dsConfig.Cluster.CloudTarget,
		}

		dsConfig.Cluster.Namespace = &targetConfig.Namespace
		dsConfig.Cluster.KubeContext = &kubeContext

		err := updateKubeConfig(kubeContext, targetConfig)
		if err != nil {
			return err
		}
	} else {
		dsConfig.Cluster = &v1.Cluster{
			CloudProvider: dsConfig.Cluster.CloudProvider,
			CloudTarget:   dsConfig.Cluster.CloudTarget,
		}

		dsConfig.Cluster.APIServer = &targetConfig.Server
		dsConfig.Cluster.Namespace = &targetConfig.Namespace
		dsConfig.Cluster.CaCert = &targetConfig.CaCert
		dsConfig.Cluster.User = &v1.ClusterUser{
			Token: &targetConfig.ServiceAccountToken,
		}
	}

	return nil
}

func updateKubeConfig(contextName string, targetConfig *generated.DevSpaceTargetConfig) error {
	config, err := kubeconfig.ReadKubeConfig(clientcmd.RecommendedHomeFile)
	if err != nil {
		return err
	}
	caCert, err := base64.StdEncoding.DecodeString(targetConfig.CaCert)
	if err != nil {
		return err
	}

	cluster := api.NewCluster()
	cluster.Server = targetConfig.Server
	cluster.CertificateAuthorityData = caCert

	authInfo := api.NewAuthInfo()
	authInfo.Token = targetConfig.ServiceAccountToken

	config.Clusters[contextName] = cluster
	config.AuthInfos[contextName] = authInfo

	// Update kube context
	context := api.NewContext()
	context.Cluster = contextName
	context.AuthInfo = contextName
	context.Namespace = targetConfig.Namespace

	config.Contexts[contextName] = context

	return kubeconfig.WriteKubeConfig(config, clientcmd.RecommendedHomeFile)
}
