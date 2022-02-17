package kubeconfig

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/loft-sh/devspace/pkg/util/fsutil"

	"gotest.tools/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestSaveLoadKubeConfig(t *testing.T) {
	t.Skip("Test not ready yet")

	dir := t.TempDir()

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
		defer func() {
			_ = fsutil.Copy("configBackup", clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename(), true)
		}()
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
	}()

	//Create object testConfig
	extensions := make(map[string]runtime.Object)
	clusters := make(map[string]*api.Cluster)
	clusters["testCluster"] = &api.Cluster{
		Server:                   "testServer",
		LocationOfOrigin:         "config",
		InsecureSkipTLSVerify:    true,
		CertificateAuthority:     "TestCA",
		CertificateAuthorityData: []byte("TestCAData"),
		Extensions:               extensions,
	}
	authInfos := make(map[string]*api.AuthInfo)
	authInfos["testAuthInfo"] = &api.AuthInfo{
		ClientCertificate:     "testCC",
		ClientCertificateData: []byte("testCCData"),
		ClientKey:             "testClientKey",
		ClientKeyData:         []byte("testClientKeyData"),
		Token:                 "testToken",
		TokenFile:             "someTokenFile",
		Impersonate:           "testImpersonate",
		ImpersonateGroups:     []string{"testIG"},
		ImpersonateUserExtra:  map[string][]string{"testIUEKey": {"testIUE"}},
		Password:              "password",
		LocationOfOrigin:      "config",
		AuthProvider: &api.AuthProviderConfig{
			Name:   "TestAuthProvider",
			Config: map[string]string{"testConfigKey": "testConfigValue"},
		},
		Exec: &api.ExecConfig{
			Command: "Do",
			Args:    []string{"something"},
			Env: []api.ExecEnvVar{
				{
					Name:  "testExecEnvVarKey",
					Value: "testExecEnvVarValue",
				},
			},
			APIVersion: "testExecVersion",
		},
		Extensions: extensions,
	}
	contexts := make(map[string]*api.Context)
	contexts["testContext"] = &api.Context{
		Cluster:          "testCluster",
		LocationOfOrigin: "config",
		AuthInfo:         "testAI",
		Namespace:        "testNS",
		Extensions:       extensions,
	}

	testConfig := &api.Config{
		Preferences: api.Preferences{
			Colors:     true,
			Extensions: extensions,
		},
		Clusters:       clusters,
		AuthInfos:      authInfos,
		Contexts:       contexts,
		Extensions:     extensions,
		CurrentContext: "testContext",
	}

	loader := NewLoader()

	err = loader.SaveConfig(testConfig)
	if err != nil {
		t.Fatalf("Error calling WriteKubeConfig: %v", err)
	}

	kubeConfig, err := loader.LoadRawConfig()
	if err != nil {
		t.Fatalf("Error calling LoadRawConfig: %v", err)
	}

	//Adapt filepaths of testConfig
	if kubeConfig != nil && kubeConfig.Clusters != nil && kubeConfig.Clusters["testCluster"] != nil && strings.HasSuffix(kubeConfig.Clusters["testCluster"].CertificateAuthority, testConfig.Clusters["testCluster"].CertificateAuthority) {
		kubePath := strings.TrimSuffix(kubeConfig.Clusters["testCluster"].CertificateAuthority, testConfig.Clusters["testCluster"].CertificateAuthority)
		testConfig.Clusters["testCluster"].CertificateAuthority = kubePath + testConfig.Clusters["testCluster"].CertificateAuthority
		testConfig.AuthInfos["testAuthInfo"].ClientCertificate = kubePath + testConfig.AuthInfos["testAuthInfo"].ClientCertificate
		testConfig.AuthInfos["testAuthInfo"].ClientKey = kubePath + testConfig.AuthInfos["testAuthInfo"].ClientKey
		testConfig.AuthInfos["testAuthInfo"].TokenFile = kubePath + testConfig.AuthInfos["testAuthInfo"].TokenFile
	} else {
		kubeConfigAsJSON, _ := json.Marshal(kubeConfig)
		t.Fatalf("Wrong Config returned: %s", string(kubeConfigAsJSON))
	}

	//Adapt filepaths of the LOO-fields
	if kubeConfig != nil && kubeConfig.Clusters != nil && kubeConfig.Clusters["testCluster"] != nil && strings.HasSuffix(kubeConfig.Clusters["testCluster"].LocationOfOrigin, testConfig.Clusters["testCluster"].LocationOfOrigin) {
		kubePath := strings.TrimSuffix(kubeConfig.Clusters["testCluster"].LocationOfOrigin, testConfig.Clusters["testCluster"].LocationOfOrigin)
		testConfig.Clusters["testCluster"].LocationOfOrigin = kubePath + testConfig.Clusters["testCluster"].LocationOfOrigin
		testConfig.Contexts["testContext"].LocationOfOrigin = kubePath + testConfig.Contexts["testContext"].LocationOfOrigin
		testConfig.AuthInfos["testAuthInfo"].LocationOfOrigin = kubePath + testConfig.AuthInfos["testAuthInfo"].LocationOfOrigin
	} else {
		kubeConfigAsJSON, _ := json.Marshal(kubeConfig)
		t.Fatalf("Wrong Config returned: %s", string(kubeConfigAsJSON))
	}

	//Compare those two
	kubeConfigAsJSON, err := json.Marshal(kubeConfig)
	if err != nil {
		t.Fatalf("Error parsing to json: %v", err)
	}
	testConfigAsJSON, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("Error parsing to json: %v", err)
	}
	assert.Equal(t, string(testConfigAsJSON), string(kubeConfigAsJSON), "Readed Config doesn't match written config")

}
