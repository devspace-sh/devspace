package dependency

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	kubeconfigutil "github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestDependency(t *testing.T) {
	dir, err := ioutil.TempDir("", "testFolder")
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

	// Delete temp folder
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

	err = fsutil.WriteToFile([]byte(""), "devspace.yaml")
	if err != nil {
		t.Fatalf("Error writing file: %v", err)
	}
	err = fsutil.WriteToFile([]byte(""), "someDir/devspace.yaml")
	if err != nil {
		t.Fatalf("Error writing file: %v", err)
	}

	dependencyTasks := []*latest.DependencyConfig{
		&latest.DependencyConfig{
			Source: &latest.SourceConfig{
				Path: ptr.String("someDir"),
			},
			Config: ptr.String("someDir/devspace.yaml"),
		},
	}

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
		ImpersonateUserExtra:  map[string][]string{"testIUEKey": []string{"testIUE"}},
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
				api.ExecEnvVar{
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

	kubeAPIConfig := &api.Config{
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

	err = kubeconfigutil.SaveConfig(kubeAPIConfig)
	if err != nil {
		t.Fatalf("Error saving kube api config: %v", err)
	}
	kubeConfig, err := kubeconfigutil.LoadRawConfig()
	if err != nil {
		t.Fatalf("Error loading raw config: %v", err)
	}
	t.Log(kubeConfig)

	testConfig := &latest.Config{
		Dependencies: &dependencyTasks,
	}
	// Create fake generated config
	generatedConfig := &generated.Config{
		ActiveConfig: "default",
		Configs: map[string]*generated.CacheConfig{
			"default": &generated.CacheConfig{
				Images: map[string]*generated.ImageCache{
					"default": &generated.ImageCache{
						Tag: "1.15", // This will be appended to nginx during deploy
					},
				},
				Dependencies: map[string]string{},
			},
		},
	}
	err = UpdateAll(&latest.Config{}, generatedConfig, true, &log.DiscardLogger{})
	if err != nil {
		t.Fatalf("Error updating all dependencies with empty config: %v", err)
	}

	err = UpdateAll(testConfig, generatedConfig, true, &log.DiscardLogger{})
	if err != nil {
		t.Fatalf("Error updating all dependencies: %v", err)
	}

	err = DeployAll(&latest.Config{}, generatedConfig, true, true, true, true, true, true, &log.DiscardLogger{})
	if err != nil {
		t.Fatalf("Error deploying all dependencies with empty config: %v", err)
	}

	err = DeployAll(testConfig, generatedConfig, true, true, true, true, true, true, &log.DiscardLogger{})
	if err != nil {
		t.Fatalf("Error deploying all dependencies: %v", err)
	}

	err = PurgeAll(&latest.Config{}, generatedConfig, true, &log.DiscardLogger{})
	if err != nil {
		t.Fatalf("Error purging all dependencies with empty config: %v", err)
	}

	err = PurgeAll(testConfig, generatedConfig, true, &log.DiscardLogger{})
	if err != nil {
		t.Fatalf("Error purging all dependencies: %v", err)
	}

}
