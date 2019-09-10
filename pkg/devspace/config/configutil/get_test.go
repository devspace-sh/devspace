package configutil

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"testing"      

	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"gopkg.in/yaml.v2"
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
	assert.Equal(t, config.Version, latest.Version, "Initialized config has wrong version")
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
	GetBaseConfig(context.Background())
	if config == nil {
		t.Fatal("Config is nil after initializing")
	}
	assert.Equal(t, config.Version, latest.Version, "Initialized config has wrong version")
	assert.Equal(t, len(config.Images), 1, "Initialized config has wrong number of images")
	assert.Equal(t, config.Images["default"].Image, "defaultImage", "Initialized config has wrong image")
}

type getConfigTestCase struct {
	name string

	files map[string] interface{}
	profile string

	expectedConfig latest.Config
	expectedPanic string
	expectedOutput string
}

func TestGetConfig(t *testing.T) {
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

	_, err = os.Stat("NotThere")
	notThereError := strings.ReplaceAll(err.Error(), "NotThere", "%s")
	_, err = ioutil.ReadFile(dir)
	isDirError := strings.ReplaceAll(err.Error(), dir, "%s")

	testCases := []getConfigTestCase{
		getConfigTestCase{
			name: "no files",
			expectedPanic: fmt.Sprintf("Couldn't find 'devspace.yaml': " + notThereError, "devspace.yaml"),
		},
		getConfigTestCase{
			name: "unparsable generated.yaml",
			files: map[string]interface{}{
				generated.ConfigPath: "unparsable",
			},
			expectedPanic: "Error loading .devspace/generated.yaml: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `unparsable` into generated.Config",
		},
		getConfigTestCase{
			name: "deprecated devspace-configs.yaml exists",
			files: map[string]interface{}{
				constants.DefaultConfigsPath: "",
			},
			expectedPanic: "devspace-configs.yaml is not supported anymore in devspace v4. Please use 'profiles' in 'devspace.yaml' instead",
		},
		getConfigTestCase{
			name: "unparsable devspace.yaml",
			files: map[string]interface{}{
				constants.DefaultConfigPath: "unparsable",
			},
			expectedPanic: "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `unparsable` into map[interface {}]interface {}",
		},
		getConfigTestCase{
			name: "devspace.yaml is a directory",
			files: map[string]interface{}{
				filepath.Join(constants.DefaultConfigPath, "someFile"): "",
			},
			expectedPanic: fmt.Sprintf(isDirError, constants.DefaultConfigPath),
		},
		getConfigTestCase{
			name: "invalid version",
			files: map[string]interface{}{
				constants.DefaultConfigPath: latest.Config{
					Version: "ThisVersionDoesNotAndWillHopefullyNeverExistEver",
				},
			},
			expectedPanic: "Unrecognized config version ThisVersionDoesNotAndWillHopefullyNeverExistEver. Please upgrade devspace with `devspace upgrade`",
		},
		getConfigTestCase{
			name: "invalid config",
			files: map[string]interface{}{
				constants.DefaultConfigPath: latest.Config{
					Version: latest.Version,
					Dev: &latest.DevConfig{
						Selectors: []*latest.SelectorConfig{
							&latest.SelectorConfig{},
						},
					},
				},
			},
			expectedPanic: "Error in config: Unnamed selector at index 0",
		},
	}

	for _, testCase := range testCases{
		testGetConfig(t, testCase)
	}
}

func testGetConfig(t *testing.T, testCase getConfigTestCase){
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
		rec := recover()
		if testCase.expectedPanic == "" {
			if rec != nil {
				t.Fatalf("Unexpected panic in testCase %s. Message: %s. Stack: %s", testCase.name, rec, string(debug.Stack()))
			}
		} else {
			if rec == nil {
				t.Fatalf("Unexpected no panic in testCase %s", testCase.name)
			} else {
				assert.Equal(t, rec, testCase.expectedPanic, "Wrong panic message in testCase %s. Stack: %s", testCase.name, string(debug.Stack()))
			}
		}
		assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)

		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}()

	for path, content := range testCase.files {
		asYAML, err := yaml.Marshal(content)
		assert.NilError(t, err, "Error parsing config to yaml in testCase %s", testCase.name)
		err = fsutil.WriteToFile(asYAML, path)
		assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{
			PanicOnExit: true,
		},
	})
	getConfigOnce = sync.Once{}
	generated.ResetConfig()
	
	GetConfig(context.Background(), testCase.profile)

	expected := testCase.expectedConfig
	assert.Equal(t, config.Version, expected.Version, "Returned context has wrong version in testCase %s", testCase.name)
}

func TestSetDevspaceRoot(t *testing.T) {
	configBackup := config
	defer func() { config = configBackup }()
	config = nil

	//Create tempDir and go into it
	dir, err := ioutil.TempDir("", "testDir")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
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

	testConfig.Dev.Selectors = []*latest.SelectorConfig{
		&latest.SelectorConfig{
			Name: "NotFound",
			Namespace: "WrongNS",
		},
		&latest.SelectorConfig{
			Name: "Found",
			Namespace: "CorrectNS",
		},
	}
	//Invalid: No selector
	selector, err = GetSelector(testConfig, "Found")
	if err != nil {
		t.Fatalf("Error getting an existent selector: %v", err)
	}
	assert.Equal(t, "CorrectNS", selector.Namespace, "Wrong selector returned")
}

func TestValidate(t *testing.T) {
	err := validate(&latest.Config{})
	if err != nil {
		t.Fatalf("Error in empty config found: %v", err)
	}

	err = validate(&latest.Config{
		Dev: &latest.DevConfig{
			Selectors: []*latest.SelectorConfig{
				&latest.SelectorConfig{},
			},
		},
	})
	if err == nil {
		t.Fatalf("No error in config with nameless selector: %v", err)
	}

	err = validate(&latest.Config{
		Dev: &latest.DevConfig{
			Ports: []*latest.PortForwardingConfig{
				&latest.PortForwardingConfig{},
			},
		},
	})
	if err == nil {
		t.Fatalf("No error in config with invalid port forwarding: %v", err)
	}

	err = validate(&latest.Config{
		Dev: &latest.DevConfig{
			Ports: []*latest.PortForwardingConfig{
				&latest.PortForwardingConfig{
					Selector: "",
				},
			},
		},
	})
	if err == nil {
		t.Fatalf("No error in config with invalid port forwarding: %v", err)
	}

	err = validate(&latest.Config{
		Dev: &latest.DevConfig{
			Sync: []*latest.SyncConfig{
				&latest.SyncConfig{},
			},
		},
	})
	if err == nil {
		t.Fatalf("No error in config with invalid sync: %v", err)
	}

	err = validate(&latest.Config{
		Hooks: []*latest.HookConfig{
			&latest.HookConfig{},
		},
	})
	if err == nil {
		t.Fatalf("No error in config with invalid hook: %v", err)
	}

	err = validate(&latest.Config{
		Images: map[string]*latest.ImageConfig{
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
		Deployments: []*latest.DeploymentConfig{
			&latest.DeploymentConfig{},
		},
	})
	if err == nil {
		t.Fatalf("No error in config with invalid deployment %v", err)
	}

	err = validate(&latest.Config{
		Deployments: []*latest.DeploymentConfig{
			&latest.DeploymentConfig{
				Name: "Invalid deployment",
			},
		},
	})
	if err == nil {
		t.Fatalf("No error in config with invalid deployment %v", err)
	}

	err = validate(&latest.Config{
		Deployments: []*latest.DeploymentConfig{
			&latest.DeploymentConfig{
				Name: "Invalid deployment",
				Helm: &latest.HelmConfig{},
			},
		},
	})
	if err == nil {
		t.Fatalf("No error in config with invalid deployment %v", err)
	}

	err = validate(&latest.Config{
		Deployments: []*latest.DeploymentConfig{
			&latest.DeploymentConfig{
				Name: "Invalid deployment",
				Kubectl: &latest.KubectlConfig{},
			},
		},
	})
	if err == nil {
		t.Fatalf("No error in config with invalid deployment %v", err)
	}

	err = validate(&latest.Config{
		Dev: &latest.DevConfig{
			Selectors: []*latest.SelectorConfig{
				&latest.SelectorConfig{
					Name: "Valid",
				},
			},
			Ports: []*latest.PortForwardingConfig{
				&latest.PortForwardingConfig{
					Selector: "mySelector",
					PortMappings: []*latest.PortMapping{},
				},
			},
			Sync: []*latest.SyncConfig{
				&latest.SyncConfig{
					Selector: "mySelector",
				},
			},
		},
		Hooks: []*latest.HookConfig{
			&latest.HookConfig{
				Command: "echo",
			},
		},
		Images: map[string]*latest.ImageConfig{
			"validImg": &latest.ImageConfig{
				Image: "someImage",
				Build: &latest.BuildConfig{
					Custom: &latest.CustomConfig{
						Command: "echo",
					},
				},
			},
		},
		Deployments: []*latest.DeploymentConfig{
			&latest.DeploymentConfig{
				Name: "Valid deployment",
				Component: &latest.ComponentConfig{},
			},
		},
	})
	if err != nil {
		t.Fatalf("Error in valid config found: %v", err)
	}
}
