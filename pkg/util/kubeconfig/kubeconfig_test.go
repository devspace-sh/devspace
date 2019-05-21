package kubeconfig

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/apimachinery/pkg/runtime"
	"gotest.tools/assert"
)

func TestGetConfigExists(t *testing.T){
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
	err = fsutil.Copy(clientcmd.RecommendedHomeFile, "configBackup", true)
	if !os.IsNotExist(err) {
		defer fsutil.Copy("configBackup", clientcmd.RecommendedHomeFile, true)
	}

	// 8. Delete temp folder
	defer os.Chdir(wdBackup)
	defer os.RemoveAll(dir)

	err = fsutil.WriteToFile([]byte(""), clientcmd.RecommendedHomeFile)
	if err != nil {
		t.Fatalf("Error writing into config file: %v", err)
	}
	assert.Equal(t, true, ConfigExists(), "Method tells that config doesn't exist despite a config file being created")

	err = os.Remove(clientcmd.RecommendedHomeFile)
	if err != nil {
		t.Fatalf("Error deleting config file: %v", err)
	}
	assert.Equal(t, false, ConfigExists(), "Method tells that config exists despite a config file being deleted")
}

func TestWriteReadKubeConfig(t *testing.T) {
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

	// 8. Delete temp folder
	defer os.Chdir(wdBackup)
	defer os.RemoveAll(dir)

	extensions := make(map[string]runtime.Object)
	clusters := make(map[string]*api.Cluster)
	clusters["testCluster"] = &api.Cluster{
		LocationOfOrigin: "testLoO",
		Server: "testServer",
		InsecureSkipTLSVerify: true,
		CertificateAuthority: "TestCA",
		CertificateAuthorityData: []byte("TestCAData"),
		Extensions: extensions,
	}
	authInfos := make(map[string]*api.AuthInfo)
	authInfos["testAuthInfo"] = &api.AuthInfo{
		LocationOfOrigin: "testLoO",
		ClientCertificate: "testCC",
		ClientCertificateData: []byte("testCCData"),
		ClientKey: "testClientKey",
		ClientKeyData: []byte("testClientKeyData"),
		Token: "testToken",
		TokenFile: "someTokenFile",
		Impersonate: "testImpersonate",
		ImpersonateGroups: []string{"testIG"},
		ImpersonateUserExtra: map[string][]string{"testIUEKey": []string{"testIUE"}},
		Password: "password",
		AuthProvider: &api.AuthProviderConfig{
			Name: "TestAuthProvider",
			Config: map[string]string{"testConfigKey": "testConfigValue"},
		},
		Exec: &api.ExecConfig{
			Command: "Do",
			Args: []string{"something"},
			Env: []api.ExecEnvVar{
				api.ExecEnvVar{
					Name: "testExecEnvVarKey",
					Value: "testExecEnvVarValue",
				},
			},
			APIVersion: "testExecVersion",
		},
		Extensions: extensions,
	}
	contexts := make(map[string]*api.Context)
	contexts["testContext"] = &api.Context{
		LocationOfOrigin: "testLoO",
		Cluster: "testCluster",
		AuthInfo: "testAI",
		Namespace: "testNS",
		Extensions: extensions,
	}

	testConfig := &api.Config{
		Kind: "testKind",
		APIVersion: "testVersion",
		Preferences: api.Preferences{
			Colors: true,
			Extensions: extensions,
		},
		Clusters: clusters,
		AuthInfos: authInfos,
		Contexts: contexts,
		Extensions: extensions,
	}

	err = WriteKubeConfig(testConfig, "someFile")
	if err != nil {
		t.Fatalf("Error calling WriteKubeConfig: %v", err)
	}

	kubeConfig, err := ReadKubeConfig("someFile")
	if err != nil {
		t.Fatalf("Error calling ReadKubeConfig: %v", err)
	}
	kubeConfigAsJSON, err := json.Marshal(kubeConfig)
	testConfigAsJSON, err := json.Marshal(testConfig)
	assert.Equal(t, string(testConfigAsJSON), string(kubeConfigAsJSON), "Readed Config doesn't match written config")

	
}
