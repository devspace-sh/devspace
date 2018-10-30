package cloud

import (
	"fmt"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/util/kubeconfig"
	"github.com/covexo/devspace/pkg/util/log"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// DevSpaceURL holds the domain name of the connected devspace
// TODO: Change this
var DevSpaceURL = ""

// UpdateOptions specifies the possible options for the update command
type UpdateOptions struct {
	UseKubeContext    bool
	SwitchKubeContext bool
	Target            string
}

// Update updates the cloud provider information if necessary
func Update(providerConfig ProviderConfig, options *UpdateOptions, log log.Logger) error {
	dsConfig := configutil.GetConfig()

	// Don't update anything if we don't use a cloud provider
	if dsConfig.Cluster == nil || dsConfig.Cluster.CloudProvider == nil || *dsConfig.Cluster.CloudProvider == "" {
		return nil
	}

	// Get selected cloud provider from config
	selectedCloudProvider := *dsConfig.Cluster.CloudProvider

	// Get provider configuration
	provider, ok := providerConfig[selectedCloudProvider]
	if ok == false {
		return fmt.Errorf("Config for cloud provider %s couldn't be found", selectedCloudProvider)
	}

	devSpaceID := ""
	if dsConfig.Cluster.Namespace != nil {
		devSpaceID = *dsConfig.Cluster.Namespace
	}
	if devSpaceID == "" && options.Target != "" {
		return fmt.Errorf("Cannot deploy to target %s without a devspace. You need to run `devspace up` beforehand", options.Target)
	}

	domain, namespace, cluster, authInfo, err := CheckAuth(provider, devSpaceID, options.Target, log)
	if err != nil {
		return err
	}

	log.Infof("Successfully logged into %s", selectedCloudProvider)
	DevSpaceURL = domain

	err = updateDevSpaceConfig(namespace, cluster, authInfo, options)
	if err != nil {
		return err
	}

	return nil
}

func updateDevSpaceConfig(namespace string, cluster *api.Cluster, authInfo *api.AuthInfo, options *UpdateOptions) error {
	dsConfig := configutil.GetConfig()
	overwriteConfig := configutil.GetOverwriteConfig()
	saveConfig := false

	// Update tiller if needed
	if dsConfig.Tiller != nil && dsConfig.Tiller.Namespace != nil {
		*dsConfig.Tiller.Namespace = namespace
	}

	// Update registry namespace if needed
	if dsConfig.InternalRegistry != nil && dsConfig.InternalRegistry.Namespace != nil {
		*dsConfig.InternalRegistry.Namespace = namespace
	}

	// Exchange cluster information
	if options.UseKubeContext {
		kubeContext := DevSpaceKubeContextName + "-" + namespace

		if dsConfig.Cluster.KubeContext == nil || *dsConfig.Cluster.KubeContext != kubeContext || dsConfig.Cluster.Namespace == nil || *dsConfig.Cluster.Namespace != namespace {
			dsConfig.Cluster = &v1.Cluster{
				CloudProvider:             dsConfig.Cluster.CloudProvider,
				CloudProviderDeployTarget: dsConfig.Cluster.CloudProviderDeployTarget,
			}

			overwriteConfig.Cluster = &v1.Cluster{
				Namespace:   &namespace,
				KubeContext: configutil.String(kubeContext),
			}

			dsConfig.Cluster.Namespace = overwriteConfig.Cluster.Namespace
			dsConfig.Cluster.KubeContext = overwriteConfig.Cluster.KubeContext

			saveConfig = true
		}

		if saveConfig || options.SwitchKubeContext {
			err := UpdateKubeConfig(kubeContext, namespace, cluster, authInfo, options.SwitchKubeContext)
			if err != nil {
				return err
			}
		}
	} else {
		if dsConfig.Cluster.APIServer == nil || *dsConfig.Cluster.APIServer != cluster.Server || dsConfig.Cluster.Namespace == nil || *dsConfig.Cluster.Namespace != namespace {
			dsConfig.Cluster = &v1.Cluster{
				CloudProvider:             dsConfig.Cluster.CloudProvider,
				CloudProviderDeployTarget: dsConfig.Cluster.CloudProviderDeployTarget,
			}

			overwriteConfig.Cluster = &v1.Cluster{
				APIServer: &cluster.Server,
				Namespace: &namespace,
				CaCert:    configutil.String(string(cluster.CertificateAuthorityData)),
				User: &v1.ClusterUser{
					Token: configutil.String(string(authInfo.Token)),
				},
			}

			dsConfig.Cluster.APIServer = overwriteConfig.Cluster.APIServer
			dsConfig.Cluster.Namespace = overwriteConfig.Cluster.Namespace
			dsConfig.Cluster.CaCert = overwriteConfig.Cluster.CaCert
			dsConfig.Cluster.User = overwriteConfig.Cluster.User

			saveConfig = true
		}
	}

	if saveConfig && options.Target == "" {
		err := configutil.SaveConfig()
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateKubeConfig adds the devspace-cloud context if necessary and switches the current context
func UpdateKubeConfig(contextName, namespace string, cluster *api.Cluster, authInfo *api.AuthInfo, switchContext bool) error {
	config, err := kubeconfig.ReadKubeConfig(clientcmd.RecommendedHomeFile)
	if err != nil {
		return err
	}

	// Switch context if necessary
	if switchContext && config.CurrentContext != contextName {
		config.CurrentContext = contextName
	}

	config.Clusters[contextName] = cluster
	config.AuthInfos[contextName] = authInfo

	// Update kube context
	context := api.NewContext()
	context.Cluster = contextName
	context.AuthInfo = contextName
	context.Namespace = namespace

	config.Contexts[contextName] = context

	return kubeconfig.WriteKubeConfig(config, clientcmd.RecommendedHomeFile)
}
