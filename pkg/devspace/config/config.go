package config

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/util/fsutil"
	"k8s.io/client-go/tools/clientcmd"

	yaml "gopkg.in/yaml.v2"
)

type ConfigInterface interface{}

const configGitignore = `logs/
private.yaml
cache.yaml
`

var workdir, _ = os.Getwd()
var configFilesLoaded = map[string][]byte{}
var configPaths = map[string]string{
	"PrivateConfig":  workdir + "/.devspace/private.yaml",
	"DevSpaceConfig": workdir + "/.devspace/config.yaml",
}

func GetConfigPath(config ConfigInterface) (string, error) {
	configType, _ := getConfigType(config)
	configPath, _ := configPaths[configType]

	return configPath, nil
}

func ConfigExists(config ConfigInterface) (bool, error) {
	configPath, _ := GetConfigPath(config)

	_, configNotFound := os.Stat(configPath)

	return (configNotFound == nil), nil
}

func LoadConfig(config ConfigInterface) error {
	configType, _ := getConfigType(config)
	loadedFile, isLoaded := configFilesLoaded[configType]

	if !isLoaded {
		loadYamlFromFile(configType)

		loadedFile, isLoaded = configFilesLoaded[configType]

		if !isLoaded {
			return errors.New("Unable to load " + configType)
		}
	}
	unmarshalErr := yaml.Unmarshal(loadedFile, config)

	if configType == "PrivateConfig" {
		privateConf, isPrivateConf := config.(*v1.PrivateConfig)

		if isPrivateConf && privateConf.Cluster.UseKubeConfig {
			LoadClusterConfig(privateConf.Cluster, false)
		}
	}
	return unmarshalErr
}

func LoadClusterConfig(config *v1.Cluster, overwriteExistingValues bool) {
	kubeconfig, kubeconfigErr := clientcmd.BuildConfigFromFlags("", filepath.Join(fsutil.GetHomeDir(), ".kube", "config"))

	if kubeconfigErr == nil {
		if len(config.ApiServer) == 0 {
			if len(kubeconfig.Host) != 0 {
				config.ApiServer = kubeconfig.Host
			}
		}

		if len(config.CaCert) == 0 {
			if len(kubeconfig.TLSClientConfig.CAData) == 0 {
				caData, caFileErr := fsutil.ReadFile(kubeconfig.TLSClientConfig.CAFile, 0)

				if caFileErr == nil {
					config.CaCert = string(caData)
				}
			} else {
				config.CaCert = string(kubeconfig.CAData)
			}
		}

		if config.User == nil {
			config.User = &v1.User{}
		}

		if len(config.User.Username) == 0 {
			config.User.Username = kubeconfig.Username
		}

		if len(config.User.ClientCert) == 0 {
			if len(kubeconfig.TLSClientConfig.CertData) == 0 {
				certData, certFileErr := fsutil.ReadFile(kubeconfig.TLSClientConfig.CertFile, 0)

				if certFileErr == nil {
					config.User.ClientCert = string(certData)
				}
			} else {
				config.User.ClientCert = string(kubeconfig.TLSClientConfig.CertData)
			}
		}

		if len(config.User.ClientKey) == 0 {
			if len(kubeconfig.TLSClientConfig.KeyData) == 0 {
				keyData, keyFileErr := fsutil.ReadFile(kubeconfig.TLSClientConfig.KeyFile, 0)

				if keyFileErr == nil {
					config.User.ClientKey = string(keyData)
				}
			} else {
				config.User.ClientKey = string(kubeconfig.TLSClientConfig.KeyData)
			}
		}
	}
}

func SaveConfig(config ConfigInterface) error {
	configType, _ := getConfigType(config)
	var currentClusterConfig v1.Cluster
	isPrivateConf := (configType == "PrivateConfig")

	if isPrivateConf {
		privateConf, isPrivateConf := config.(*v1.PrivateConfig)
		currentClusterConfig = *privateConf.Cluster

		if isPrivateConf && privateConf.Cluster.UseKubeConfig {
			privateConf.Cluster.ApiServer = ""
			privateConf.Cluster.CaCert = ""
			privateConf.Cluster.User = nil
		}
	}
	yamlString, yamlErr := yaml.Marshal(config)

	if isPrivateConf {
		privateConf, _ := config.(*v1.PrivateConfig)

		privateConf.Cluster = &currentClusterConfig
	}

	if yamlErr != nil {
		return yamlErr
	}
	configFilesLoaded[configType] = yamlString

	return saveYamlToFile(configType)
}

func getConfigType(config ConfigInterface) (string, error) {
	configGolangType := reflect.TypeOf(config).String()
	configType := strings.Split(configGolangType, ".")[1]

	return configType, nil
}

func loadYamlFromFile(configType string) error {
	yamlFileContent, err := ioutil.ReadFile(configPaths[configType])

	if err != nil {
		return err
	}
	configFilesLoaded[configType] = yamlFileContent

	return nil
}

func saveYamlToFile(configType string) error {
	os.MkdirAll(filepath.Dir(configPaths[configType]), os.ModePerm)

	if configType == "PrivateConfig" {
		fsutil.WriteToFile([]byte(configGitignore), filepath.Join(filepath.Dir(configPaths[configType]), ".gitignore"))
	}
	return ioutil.WriteFile(configPaths[configType], configFilesLoaded[configType], os.ModePerm)
}
