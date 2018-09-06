package configutil

import (
	"io/ioutil"
	"path/filepath"

	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/util/fsutil"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/client-go/tools/clientcmd"
)

func loadConfig(config *v1.Config, path string) error {
	yamlFileContent, err := ioutil.ReadFile(workdir + path)

	if err != nil {
		return err
	}
	return yaml.Unmarshal(yamlFileContent, config)
}

//LoadClusterConfig loads the config for a kubernetes cluster
func loadClusterConfig(config *v1.Cluster, overwriteExistingValues bool) {
	kubeconfig, kubeconfigErr := clientcmd.BuildConfigFromFlags("", filepath.Join(fsutil.GetHomeDir(), ".kube", "config"))

	if kubeconfigErr == nil {
		if config.APIServer == nil {
			if len(kubeconfig.Host) != 0 {
				config.APIServer = String(kubeconfig.Host)
			}
		}

		if config.CaCert == nil {
			if len(kubeconfig.TLSClientConfig.CAData) == 0 {
				caData, caFileErr := fsutil.ReadFile(kubeconfig.TLSClientConfig.CAFile, 0)

				if caFileErr == nil {
					config.CaCert = String(string(caData))
				}
			} else {
				config.CaCert = String(string(kubeconfig.CAData))
			}
		}

		if config.User == nil {
			config.User = &v1.User{}
		}

		if config.User.Username == nil {
			config.User.Username = String(kubeconfig.Username)
		}

		if config.User.ClientCert == nil {
			if len(kubeconfig.TLSClientConfig.CertData) == 0 {
				certData, certFileErr := fsutil.ReadFile(kubeconfig.TLSClientConfig.CertFile, 0)

				if certFileErr == nil {
					config.User.ClientCert = String(string(certData))
				}
			} else {
				config.User.ClientCert = String(string(kubeconfig.TLSClientConfig.CertData))
			}
		}

		if config.User.ClientKey == nil {
			if len(kubeconfig.TLSClientConfig.KeyData) == 0 {
				keyData, keyFileErr := fsutil.ReadFile(kubeconfig.TLSClientConfig.KeyFile, 0)

				if keyFileErr == nil {
					config.User.ClientKey = String(string(keyData))
				}
			} else {
				config.User.ClientKey = String(string(kubeconfig.TLSClientConfig.KeyData))
			}
		}
	}
}
