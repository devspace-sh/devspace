package cloud

import (
	"fmt"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/util/kubeconfig"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// UpdateOptions specifies the possible options for the update command
type UpdateOptions struct {
	UseKubeContext    bool
	SwitchKubeContext bool
	CloudProvider     string
	Target            string
}

// Update updates the cloud provider information if necessary
func Update(providerConfig ProviderConfig, dsConfig *v1.Config, options *UpdateOptions) error {
	// Don't update anything if we don't use a cloud provider
	if options.CloudProvider == "" {
		return nil
	}

	provider, ok := providerConfig[options.CloudProvider]
	if ok == false {
		return fmt.Errorf("Config for cloud provider %s couldn't be found", options.CloudProvider)
	}

	devSpaceID := ""
	if dsConfig.Cluster.Namespace != nil {
		devSpaceID = *dsConfig.Cluster.Namespace
	}
	if devSpaceID == "" && options.Target != "" {
		return fmt.Errorf("Cannot deploy to target %s without a devspace. You need to run `devspace up` beforehand", options.Target)
	}

	namespace, cluster, authInfo, err := CheckAuth(provider, devSpaceID, options.Target)
	if err != nil {
		return err
	}

	// Update tiller if needed
	if dsConfig.Tiller != nil {
		dsConfig.Tiller.Namespace = &namespace
	}

	// Update registry namespace if needed
	if dsConfig.InternalRegistry != nil {
		dsConfig.InternalRegistry.Namespace = &namespace
	}

	if options.UseKubeContext {
		kubeContext := DevSpaceKubeContextName + "-" + namespace

		err = UpdateKubeConfig(kubeContext, namespace, cluster, authInfo, options.SwitchKubeContext)
		if err != nil {
			return err
		}

		dsConfig.Cluster.Namespace = &namespace
		dsConfig.Cluster.KubeContext = configutil.String(kubeContext)
	} else {
		dsConfig.Cluster.APIServer = &cluster.Server
		dsConfig.Cluster.Namespace = &namespace
		dsConfig.Cluster.CaCert = configutil.String(string(cluster.CertificateAuthorityData))

		dsConfig.Cluster.User = &v1.ClusterUser{
			ClientCert: configutil.String(string(authInfo.ClientCertificateData)),
			ClientKey:  configutil.String(string(authInfo.ClientKeyData)),
			Token:      configutil.String(string(authInfo.Token)),
		}
	}

	return err
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
