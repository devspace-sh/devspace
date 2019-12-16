package cloud

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"

	"k8s.io/client-go/tools/clientcmd"

	"gotest.tools/assert"
)

func TestGetKubeContextNameFromSpace(t *testing.T) {
	assert.Equal(t, GetKubeContextNameFromSpace("space:Name", "provider.Name"), DevSpaceKubeContextName+"-provider-name-space-name", "Wrong KubeContextName returned")
}

func TestUpdateKubeConfig(t *testing.T) {
	testProvider := &provider{Provider: latest.Provider{Host: "app.devspace.cloud"}}
	err := testProvider.UpdateKubeConfig("", &latest.ServiceAccount{CaCert: "Undecodable"}, 1, false)
	assert.Error(t, err, "illegal base64 data at input byte 8", "No or wrong error when trying to update kube config with an undecodable cacert in the serviceaccount")

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
	if !os.IsNotExist(err) {
		os.Remove(clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename())
		defer fsutil.Copy("configBackup", clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename(), true)
	} else if err != nil {
		defer os.Remove(clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename())
	} else {
		t.Fatalf("Error making backup file: %v", err)
	}

	//Delete temp folder
	defer func() {
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}()

	err = testProvider.UpdateKubeConfig("someContext", &latest.ServiceAccount{
		CaCert:    "",
		Namespace: "someNamespace",
		Server:    "someServer",
		Token:     "someToken",
	}, 1, true)
	assert.NilError(t, err, "Error when updating kube config with a vailid serviceaccount")
	config, err := kubeconfig.LoadRawConfig()
	assert.NilError(t, err, "Error loading kubeconfig")
	assert.Equal(t, config.Contexts["someContext"].Cluster, "someContext", "KubeConfig badly saved")
	assert.Equal(t, config.Contexts["someContext"].AuthInfo, "someContext", "KubeConfig badly saved")
	assert.Equal(t, config.Contexts["someContext"].Namespace, "someNamespace", "KubeConfig badly saved")
	assert.Equal(t, config.Clusters["someContext"].Server, "someServer", "KubeConfig badly saved")
	assert.Equal(t, len(config.Clusters["someContext"].CertificateAuthorityData), 0, "KubeConfig badly saved")
	assert.Equal(t, config.AuthInfos["someContext"].Exec.Command, "devspace", "KubeConfig badly saved")
}
