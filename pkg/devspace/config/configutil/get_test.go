package configutil

import (
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	
	"k8s.io/client-go/tools/clientcmd/api"

	"gotest.tools/assert"
)

func TestConfigExists(t *testing.T) {
	configBackup := config
	SetFakeConfig(&latest.Config{})
	defer func() { config = configBackup }()

	//Create tempDir and go into it
	dir, err := ioutil.TempDir("", "testDir")
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

	// Delete temp folder after test
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

	assert.Equal(t, ConfigExists(), true, "Config doesn't exist despite being set directly in cache")

	config = nil
	fsutil.WriteToFile([]byte(""), constants.DefaultConfigPath)
	assert.Equal(t, ConfigExists(), true, "Config doesn't exist despite being set in devspace.yaml")

	err = os.Remove(constants.DefaultConfigPath)
	if err != nil {
		t.Fatalf("Error removing file: %v", err)
	}
	fsutil.WriteToFile([]byte(""), constants.DefaultConfigsPath)
	assert.Equal(t, ConfigExists(), true, "Config doesn't exist despite being set in devspace-configs.yaml")

	err = os.Remove(constants.DefaultConfigsPath)
	if err != nil {
		t.Fatalf("Error removing file: %v", err)
	}
	fsutil.WriteToFile([]byte(""), ".devspace/config.yaml")
	assert.Equal(t, ConfigExists(), true, "Config doesn't exist despite being set in .devspace/config.yaml")

	err = os.Remove(".devspace/config.yaml")
	if err != nil {
		t.Fatalf("Error removing file: %v", err)
	}
	fsutil.WriteToFile([]byte(""), ".devspace/configs.yaml")
	assert.Equal(t, ConfigExists(), true, "Config doesn't exist despite being set in .devspace/configs.yaml")

	err = os.Remove(".devspace/configs.yaml")
	if err != nil {
		t.Fatalf("Error removing file: %v", err)
	}
	assert.Equal(t, ConfigExists(), false, "Config exists despite being unset in every way")
}

func TestInitConfig(t *testing.T) {
	configBackup := config
	defer func() { config = configBackup }()
	config = nil

	getConfigOnce = sync.Once{}
	InitConfig()
	if config == nil {
		t.Fatal("Config is nil after initializing")
	}
	assert.Equal(t, *config.Version, latest.Version, "Initialized config has wrong version")
}

func TestGetBaseConfig(t *testing.T) {
	configBackup := config
	defer func() { config = configBackup }()
	config = nil

	//Create tempDir and go into it
	dir, err := ioutil.TempDir("", "testDir")
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

	// Delete temp folder after test
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

	fsutil.WriteToFile([]byte(`version: v1beta2
cluster:
  kubeContext: someKubeContext
  namespace: someNS
images:
  default:
    image: defaultImage
deployments:
- name: default
  helm:
    chart:
      name: ./chart
hooks:
- command: echo
dev:
  selectors:
  - name: someSelector
  overrideImages:
  - name: default
    entrypoint:
    - sleep
    - "999999999999"
  ports:
  - labelSelector:
      app.kubernetes.io/component: default
    forward:
    - port: 3000
  sync:
  - labelSelector:
      app.kubernetes.io/component: default
    excludePaths:
    - node_modules
`), constants.DefaultConfigPath)

	getConfigOnce = sync.Once{}
	validateOnce = sync.Once{}
	GetBaseConfig()
	if config == nil {
		t.Fatal("Config is nil after initializing")
	}
	assert.Equal(t, *config.Version, latest.Version, "Initialized config has wrong version")
	assert.Equal(t, *config.Cluster.KubeContext, "someKubeContext", "Initialized config has wrong kubeContext of cluster")
	assert.Equal(t, *config.Cluster.Namespace, "someNS", "Initialized config has wrong namespace of cluster")
	assert.Equal(t, len(*config.Images), 1, "Initialized config has wrong number of images")
	assert.Equal(t, *(*config.Images)["default"].Image, "defaultImage", "Initialized config has wrong image")
}

func TestGetConfig(t *testing.T) {
	configBackup := config
	defer func() { config = configBackup }()
	config = nil

	//Create tempDir and go into it
	dir, err := ioutil.TempDir("", "testDir")
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

	// Delete temp folder after test
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

	configString := `version: v1beta2
cluster:
  kubeContext: someKubeContext
  namespace: someNS
images:
  default:
    image: defaultImage
deployments:
- name: default
  helm:
    chart:
      name: ./chart
hooks:
- command: echo
dev:
  selectors:
  - name: someSelector
  overrideImages:
  - name: default
    entrypoint:
    - sleep
    - "999999999999"
  ports:
  - labelSelector:
      app.kubernetes.io/component: default
    forward:
    - port: 3000
  sync:
  - labelSelector:
      app.kubernetes.io/component: default
    excludePaths:
    - node_modules
`

	fsutil.WriteToFile([]byte(configString), constants.DefaultConfigPath)

	getConfigOnce = sync.Once{}
	validateOnce = sync.Once{}
	GetConfig()
	if config == nil {
		t.Fatal("Config is nil after initializing")
	}
	assert.Equal(t, *config.Version, latest.Version, "Initialized config has wrong version")
	assert.Equal(t, *config.Cluster.KubeContext, "someKubeContext", "Initialized config has wrong kubeContext of cluster")
	assert.Equal(t, *config.Cluster.Namespace, "someNS", "Initialized config has wrong namespace of cluster")
	assert.Equal(t, len(*config.Images), 1, "Initialized config has wrong number of images")
	assert.Equal(t, *(*config.Images)["default"].Image, "defaultImage", "Initialized config has wrong image")

	SetFakeConfig(&latest.Config{})

	configsContent := `default:
  vars:
    data:
      - name: hello
  config:
    path: devspace.yaml
  overrides:
    - data:
        dev:
          overrideImages:
            - name: service-image-1
              entrypoint:
                - sleep
                - "9999999999"
          terminal:
            labelSelector:
              app.kubernetes.io/component: service-1
          ports:
            - labelSelector:
                app.kubernetes.io/component: service-1
              forward:
                - port: 8080
          sync:
            - labelSelector:
                app.kubernetes.io/component: service-1
              localSubPath: ./service1
# Use the config with 'devspace use config dev-service1'
dev-service1:
  config:
    path: devspace.yaml
  # Overrides defined overridden fields in the original config
  # You can specify multiple overrides which are applied in the order
  # you specify them. Array types are completely overriden and maps will be merged
  overrides:
    - data:
        dev:
          overrideImages:
            - name: service-image-1
              entrypoint:
                - sleep
                - "9999999999"
          terminal:
            labelSelector:
              app.kubernetes.io/component: service-1
          ports:
            - labelSelector:
                app.kubernetes.io/component: service-1
              forward:
                - port: 8080
          sync:
            - labelSelector:
                app.kubernetes.io/component: service-1
              localSubPath: ./service1`
	fsutil.WriteToFile([]byte(configsContent), constants.DefaultConfigsPath)

	survey.SetNextAnswer("world")
	getConfigOnce = sync.Once{}
	validateOnce = sync.Once{}
	GetConfig()
	if config == nil {
		t.Fatal("Config is nil after initializing")
	}

	assert.Equal(t, *config.Version, latest.Version, "Initialized config has wrong version")
	assert.Equal(t, *config.Cluster.KubeContext, "someKubeContext", "Initialized config has wrong kubeContext of cluster")
	assert.Equal(t, *config.Cluster.Namespace, "someNS", "Initialized config has wrong namespace of cluster")
	assert.Equal(t, len(*config.Images), 1, "Initialized config has wrong number of images")
	assert.Equal(t, *(*config.Images)["default"].Image, "defaultImage", "Initialized config has wrong image")

	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		t.Fatalf("Error loading generated config: %v", err)
	}
	assert.Equal(t, generatedConfig.GetActive().Vars["hello"], "world", "Vars not initialized")
}

func TestSetSetDevspaceRoot(t *testing.T) {
	configBackup := config
	defer func() { config = configBackup }()
	config = nil

	//Create tempDir and go into it
	dir, err := ioutil.TempDir("", "testDir")
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

	// Delete temp folder after test
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

	//No config found
	configExists, err := SetDevSpaceRoot()
	if err != nil {
		t.Fatalf("Error setting DevSpaceRoot: %v", err)
	}
	assert.Equal(t, configExists, false, "Not existent config detected")
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting working directory: %v", err)
	}
	assert.Equal(t, currentDir, dir, "Dir changed without a config found")

	//Config found in cuurent dir
	fsutil.WriteToFile([]byte(""), constants.DefaultConfigPath)
	configExists, err = SetDevSpaceRoot()
	if err != nil {
		t.Fatalf("Error setting DevSpaceRoot: %v", err)
	}
	assert.Equal(t, configExists, true, "Existent config not detected")
	currentDir, err = os.Getwd()
	if err != nil {
		t.Fatalf("Error getting working directory: %v", err)
	}
	assert.Equal(t, currentDir, dir, "Dir changed with config found in current dir")

	//Config found in parent dir
	err = os.Mkdir("SomeSubdir", 0755)
	if err != nil {
		t.Fatalf("Error making temporary subdir: %v", err)
	}
	err = os.Chdir("SomeSubdir")
	if err != nil {
		t.Fatalf("Error changing wd to temporary subdir: %v", err)
	}
	configExists, err = SetDevSpaceRoot()
	if err != nil {
		t.Fatalf("Error setting DevSpaceRoot: %v", err)
	}
	assert.Equal(t, configExists, true, "Existent config not detected")
	currentDir, err = os.Getwd()
	if err != nil {
		t.Fatalf("Error getting working directory: %v", err)
	}
	assert.Equal(t, currentDir, dir, "Dir unchanged with config found in parent dir")

}

func TestSelector(t *testing.T){
	testConfig := &latest.Config{
		Dev: &latest.DevConfig{},
	}

	//Invalid: No selector
	selector, err := GetSelector(testConfig, "selectorName")
	if err == nil {
		t.Fatal("No Error getting a non existent selector")
	}
	if selector != nil {
		t.Fatal("Selector returned without a selector being found")
	}

	testConfig.Dev.Selectors = &[]*latest.SelectorConfig{
		&latest.SelectorConfig{
			Name: ptr.String("NotFound"),
			Namespace: ptr.String("WrongNS"),
		},
		&latest.SelectorConfig{
			Name: ptr.String("Found"),
			Namespace: ptr.String("CorrectNS"),
		},
	}
	//Invalid: No selector
	selector, err = GetSelector(testConfig, "Found")
	if err != nil {
		t.Fatalf("Error getting an existent selector: %v", err)
	}
	assert.Equal(t, "CorrectNS", *selector.Namespace, "Wrong selector returned")
}

func TestGetDefaultNamespace(t *testing.T){
	namespace, err := GetDefaultNamespace(&latest.Config{
		Cluster: &latest.Cluster{
			Namespace: ptr.String("PresetNamespace"),
		},
	})
	if err != nil{
		t.Fatalf("Error getting default namespace from config directly, %v", err)
	}
	assert.Equal(t, namespace, "PresetNamespace", "Wrong preset namespace returned")	
	
	testConfig := &api.Config{
		Contexts: map[string]*api.Context{
			"contextFromConfig": &api.Context{
				Namespace: "contextFromConfigNS",
			},
			"contextFromKubeConfig": &api.Context{
				Namespace: "contextFromKubeConfigNS",
			},
			"nilNamespaceContext": &api.Context{},
		},
		CurrentContext: "contextFromKubeConfig",
	}

	err = kubeconfig.SaveConfig(testConfig)
	if err != nil {
		t.Fatalf("Error saving kubeConfig: %v", err)
	}

	namespace, err = GetDefaultNamespace(&latest.Config{
		Cluster: &latest.Cluster{
			KubeContext: ptr.String("contextFromConfig"),
		},
	})
	if err != nil{
		t.Fatalf("Error getting default namespace from config's context, %v", err)
	}
	assert.Equal(t, namespace, "contextFromConfigNS", "Wrong preset namespace returned")

	namespace, err = GetDefaultNamespace(&latest.Config{
		Cluster: &latest.Cluster{
			KubeContext: ptr.String("nilNamespaceContext"),
		},
	})
	if err != nil{
		t.Fatalf("Error getting nil namespace from config's context, %v", err)
	}
	assert.Equal(t, namespace, "default", "Wrong preset namespace returned")	
}

func TestValidate(t *testing.T) {
	err := validate(&latest.Config{})
	if err != nil {
		t.Fatalf("Error in empty config found: %v", err)
	}

	err = validate(&latest.Config{
		Dev: &latest.DevConfig{
			Selectors: &[]*latest.SelectorConfig{
				&latest.SelectorConfig{},
			},
		},
	})
	if err == nil {
		t.Fatalf("No error in config with nameless selector: %v", err)
	}

	err = validate(&latest.Config{
		Dev: &latest.DevConfig{
			Ports: &[]*latest.PortForwardingConfig{
				&latest.PortForwardingConfig{},
			},
		},
	})
	if err == nil {
		t.Fatalf("No error in config with invalid port forwarding: %v", err)
	}

	err = validate(&latest.Config{
		Dev: &latest.DevConfig{
			Ports: &[]*latest.PortForwardingConfig{
				&latest.PortForwardingConfig{
					Selector: ptr.String(""),
				},
			},
		},
	})
	if err == nil {
		t.Fatalf("No error in config with invalid port forwarding: %v", err)
	}

	err = validate(&latest.Config{
		Dev: &latest.DevConfig{
			Sync: &[]*latest.SyncConfig{
				&latest.SyncConfig{},
			},
		},
	})
	if err == nil {
		t.Fatalf("No error in config with invalid sync: %v", err)
	}

	err = validate(&latest.Config{
		Dev: &latest.DevConfig{
			OverrideImages: &[]*latest.ImageOverrideConfig{
				&latest.ImageOverrideConfig{},
			},
		},
	})
	if err == nil {
		t.Fatalf("No error in config with invalid imageOverrideConfig: %v", err)
	}

	err = validate(&latest.Config{
		Hooks: &[]*latest.HookConfig{
			&latest.HookConfig{},
		},
	})
	if err == nil {
		t.Fatalf("No error in config with invalid hook: %v", err)
	}

	err = validate(&latest.Config{
		Images: &map[string]*latest.ImageConfig{
			"invalidImg": &latest.ImageConfig{
				Build: &latest.BuildConfig{
					Custom: &latest.CustomConfig{},
				},
			},
		},
	})
	if err == nil {
		t.Fatalf("No error in config with invalid image config: %v", err)
	}

	err = validate(&latest.Config{
		Deployments: &[]*latest.DeploymentConfig{
			&latest.DeploymentConfig{},
		},
	})
	if err == nil {
		t.Fatalf("No error in config with invalid deployment %v", err)
	}

	err = validate(&latest.Config{
		Deployments: &[]*latest.DeploymentConfig{
			&latest.DeploymentConfig{
				Name: ptr.String("Invalid deployment"),
			},
		},
	})
	if err == nil {
		t.Fatalf("No error in config with invalid deployment %v", err)
	}

	err = validate(&latest.Config{
		Deployments: &[]*latest.DeploymentConfig{
			&latest.DeploymentConfig{
				Name: ptr.String("Invalid deployment"),
				Helm: &latest.HelmConfig{},
			},
		},
	})
	if err == nil {
		t.Fatalf("No error in config with invalid deployment %v", err)
	}

	err = validate(&latest.Config{
		Deployments: &[]*latest.DeploymentConfig{
			&latest.DeploymentConfig{
				Name: ptr.String("Invalid deployment"),
				Kubectl: &latest.KubectlConfig{},
			},
		},
	})
	if err == nil {
		t.Fatalf("No error in config with invalid deployment %v", err)
	}

	err = validate(&latest.Config{
		Dev: &latest.DevConfig{
			Selectors: &[]*latest.SelectorConfig{
				&latest.SelectorConfig{
					Name: ptr.String("Valid"),
				},
			},
			Ports: &[]*latest.PortForwardingConfig{
				&latest.PortForwardingConfig{
					Selector: ptr.String(""),
					PortMappings: &[]*latest.PortMapping{},
				},
			},
			Sync: &[]*latest.SyncConfig{
				&latest.SyncConfig{
					Selector: ptr.String(""),
				},
			},
			OverrideImages: &[]*latest.ImageOverrideConfig{
				&latest.ImageOverrideConfig{
					Name: ptr.String("Valid"),
				},
			},
		},
		Hooks: &[]*latest.HookConfig{
			&latest.HookConfig{
				Command: ptr.String("echo"),
			},
		},
		Images: &map[string]*latest.ImageConfig{
			"validImg": &latest.ImageConfig{
				Build: &latest.BuildConfig{
					Custom: &latest.CustomConfig{
						Command: ptr.String("echo"),
					},
				},
			},
		},
		Deployments: &[]*latest.DeploymentConfig{
			&latest.DeploymentConfig{
				Name: ptr.String("Valid deployment"),
				Component: &latest.ComponentConfig{},
			},
		},
	})
	if err != nil {
		t.Fatalf("Error in valid config found: %v", err)
	}
}
