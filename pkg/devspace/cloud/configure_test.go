package cloud

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"k8s.io/client-go/tools/clientcmd"

	"gotest.tools/assert"
)

func TestGetProvider(t *testing.T) {
	loadedConfigOnce.Do(func() {})
	loadedConfig = ProviderConfig{}

	_, err := GetProvider(ptr.String("Doesn'tExist"), &log.DiscardLogger{})
	assert.Error(t, err, "Cloud provider not found! Did you run `devspace add provider [url]`? Existing cloud providers: ", "Wrong or not error returned when getting a non-existent provider")

	loadedConfig["Exists"] = &Provider{
		Key: "someKey",
	}
	loadedConfig["SecoundOption"] = &Provider{}
	survey.SetNextAnswer("Exists")

	provider, err := GetProvider(nil, &log.DiscardLogger{})
	assert.NilError(t, err, "Error getting valid logged in provider")
	assert.Equal(t, loadedConfig["Exists"], provider, "Srong provider returned")
	assert.Equal(t, false, provider.ClusterKey == nil, "ClusterKey of provider not set")
}

func TestGetKubeContextNameFromSpace(t *testing.T) {
	assert.Equal(t, GetKubeContextNameFromSpace("space:Name", "provider.Name"), DevSpaceKubeContextName+"-provider-name-space-name", "Wrong KubeContextName returned")
}

func TestUpdateKubeConfig(t *testing.T) {
	loadedConfigOnce.Do(func() {})
	loadedConfig = ProviderConfig{}

	err := UpdateKubeConfig("", &ServiceAccount{CaCert: "Undecodable"}, false)
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

	err = UpdateKubeConfig("someContext", &ServiceAccount{
		CaCert:    "",
		Namespace: "someNamespace",
		Server:    "someServer",
		Token:     "someToken",
	}, true)
	assert.NilError(t, err, "Error when updating kube config with a vailid serviceaccount")
	config, err := kubeconfig.LoadRawConfig()
	assert.NilError(t, err, "Error loading kubeconfig")
	assert.Equal(t, config.Contexts["someContext"].Cluster, "someContext", "KubeConfig badly saved")
	assert.Equal(t, config.Contexts["someContext"].AuthInfo, "someContext", "KubeConfig badly saved")
	assert.Equal(t, config.Contexts["someContext"].Namespace, "someNamespace", "KubeConfig badly saved")
	assert.Equal(t, config.Clusters["someContext"].Server, "someServer", "KubeConfig badly saved")
	assert.Equal(t, len(config.Clusters["someContext"].CertificateAuthorityData), 0, "KubeConfig badly saved")
	assert.Equal(t, config.AuthInfos["someContext"].Token, "someToken", "KubeConfig badly saved")
}
