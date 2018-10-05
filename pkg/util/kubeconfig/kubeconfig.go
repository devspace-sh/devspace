package kubeconfig

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"
)

// ReadKubeConfig reads the kube config from the specified filename or returns a new Config object if not found
func ReadKubeConfig(filename string) (*api.Config, error) {
	data, err := ioutil.ReadFile(filename)
	if os.IsNotExist(err) {
		return api.NewConfig(), nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "Error reading file %q", filename)
	} else if len(data) == 0 {
		return api.NewConfig(), nil
	}

	// decode config, empty if no bytes
	unconvertedConfig, _, err := latest.Codec.Decode(data, nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "Error decoding config from data: %s", string(data))
	}

	config := unconvertedConfig.(*api.Config)

	// initialize nil maps
	if config.AuthInfos == nil {
		config.AuthInfos = map[string]*api.AuthInfo{}
	}
	if config.Clusters == nil {
		config.Clusters = map[string]*api.Cluster{}
	}
	if config.Contexts == nil {
		config.Contexts = map[string]*api.Context{}
	}

	return config, nil
}

// GetCurrentContext retrieves the current context from the kube file
func GetCurrentContext() (string, error) {
	config, err := ReadKubeConfig(clientcmd.RecommendedHomeFile)
	if err != nil {
		return "", err
	}

	return config.CurrentContext, nil
}

// WriteKubeConfig writes the kube config back to the specified filename
func WriteKubeConfig(config *api.Config, filename string) error {
	// encode config to YAML
	data, err := runtime.Encode(latest.Codec, config)
	if err != nil {
		return errors.Errorf("could not write to '%s': failed to encode config: %v", filename, err)
	}

	// create parent dir if doesn't exist
	dir := filepath.Dir(filename)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return errors.Wrapf(err, "Error creating directory: %s", dir)
	}

	// write with restricted permissions
	if err := ioutil.WriteFile(filename, data, 0600); err != nil {
		return errors.Wrapf(err, "Error writing file %s", filename)
	}

	return nil
}
