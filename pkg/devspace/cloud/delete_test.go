package cloud

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"gotest.tools/assert"
)

func TestDeleteCluster(t *testing.T) {
	provider := &Provider{}
	err := provider.DeleteCluster(&latest.Cluster{}, true, true)
	assert.Error(t, err, "get token: Provider has no key specified", "Wrong or no error when trying to delete a cluster without a token")
}

func TestDeleteSpace(t *testing.T) {
	provider := &Provider{}
	err := provider.DeleteSpace(&latest.Space{Cluster: &latest.Cluster{}})
	assert.Error(t, err, "get token: Provider has no key specified", "Wrong or no error when trying to delete a space without a token")
}

func TestDeleteKubeContext(t *testing.T) {
	dir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}

	//Make Backup of config file
	err = fsutil.Copy(clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename(), "configBackup", true)
	didConfigExist := !os.IsNotExist(err)

	//Delete temp folder
	defer func() {
		if didConfigExist {
			err = fsutil.Copy("configBackup", clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename(), true)
		} else {
			err = os.Remove(clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename())
		}
		assert.NilError(t, err, "Error restoring config")

		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}()

	kubeContext := GetKubeContextNameFromSpace("space.Name", "space.ProviderName")
	config, err := kubeconfig.LoadRawConfig()
	assert.NilError(t, err, "Error loading kubeConfig")
	config.CurrentContext = kubeContext
	config.Contexts = map[string]*api.Context{
		"otherContext": &api.Context{},
		kubeContext:    &api.Context{},
	}
	config.Clusters = map[string]*api.Cluster{
		kubeContext: &api.Cluster{},
	}
	config.AuthInfos = map[string]*api.AuthInfo{
		kubeContext: &api.AuthInfo{},
	}
	kubeconfig.SaveConfig(config)

	err = DeleteKubeContext(&latest.Space{Name: "space.Name", ProviderName: "space.ProviderName"})
	assert.NilError(t, err, "Error deleting kube context")

	config, err = kubeconfig.LoadRawConfig()
	assert.NilError(t, err, "Error loading kubeConfig")
	assert.Equal(t, len(config.Contexts), 1, "kube context not correctly deleted")
	assert.Equal(t, len(config.Clusters), 1, "kube context not correctly deleted")
	assert.Equal(t, len(config.AuthInfos), 1, "kube context not correctly deleted")
	assert.Equal(t, config.CurrentContext, "otherContext", "kube context not correctly deleted")

	err = DeleteKubeContext(&latest.Space{Name: "space.Name", ProviderName: "space.ProviderName"})
	assert.NilError(t, err, "Error deleting already deleted kube context")
}
