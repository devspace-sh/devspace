package cloud

import (
	"encoding/base64"
	"regexp"
	"strconv"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"k8s.io/client-go/tools/clientcmd/api"
)

// SpaceNameValidationRegEx is the sapace name validation regex
var SpaceNameValidationRegEx = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9-]{1,30}[a-zA-Z0-9]$")

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
func (p *provider) UpdateKubeConfig(contextName string, serviceAccount *latest.ServiceAccount, spaceID int, setActive bool) error {
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
		Args:       []string{"use", "space", "--provider", p.Name, "--space-id", strconv.Itoa(spaceID), "--get-token", "--silent"},
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
