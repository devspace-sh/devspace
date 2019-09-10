package configutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	yaml "gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

func TestRestoreVars(t *testing.T) {
	testConfig := &latest.Config{
		Version: latest.Version,
	}

	defer func() { LoadedVars = map[string]string{} }()
	LoadedVars[".version"] = "someVersion"

	resultConfig, err := RestoreVars(testConfig)

	assert.NilError(t, err, "Error Restoring Vars")
	assert.Equal(t, "someVersion", resultConfig.Version, "Loaded var not correctly applied")
}

type saveLoadedConfigTestCase struct {
	name string

	config *latest.Config
	files  map[string]interface{}

	expectedOutput  string
	expectedPanic   string
	expectedErr     string
	expectedContent string
}

func TestSaveLoadedConfig(t *testing.T) {
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

	_, err = ioutil.ReadFile(dir)
	isDirError := strings.ReplaceAll(err.Error(), dir, "%s")

	testCases := []saveLoadedConfigTestCase{
		saveLoadedConfigTestCase{
			name: "1 Profile",
			config: &latest.Config{
				Profiles: []*latest.ProfileConfig{
					&latest.ProfileConfig{},
				},
			},
			expectedErr: "Cannot save when a profile is applied",
		},
		saveLoadedConfigTestCase{
			name:   "devspace.yaml is a dir",
			config: &latest.Config{},
			files: map[string]interface{}{
				filepath.Join(constants.DefaultConfigPath, "someFile"): "",
			},
			expectedErr: fmt.Sprintf("restore vars: " + isDirError, constants.DefaultConfigPath),
		},
		saveLoadedConfigTestCase{
			name:   "unparsable devspace.yaml",
			config: &latest.Config{},
			files: map[string]interface{}{
				constants.DefaultConfigPath: "unparsable",
			},
			expectedErr: "restore vars: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `unparsable` into map[interface {}]interface {}",
		},
		saveLoadedConfigTestCase{
			name: "save with success",
			config: &latest.Config{
				Version: latest.Version,
			},
			files: map[string]interface{}{
				constants.DefaultConfigPath: &latest.Config{
					Dev: &latest.DevConfig{},
				},
			},
			expectedContent: "version: v1beta3\ndev: {}\n",
		},
	}

	for _, testCase := range testCases {
		testSaveLoadedConfig(t, testCase)
	}
}

func testSaveLoadedConfig(t *testing.T, testCase saveLoadedConfigTestCase) {
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

	config = testCase.config

	err = SaveLoadedConfig()
	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s", testCase.name)
		content, err := fsutil.ReadFile(constants.DefaultConfigPath, -1)
		assert.NilError(t, err, "Error reading devspace.yaml in testCase %s", testCase.name)
		assert.Equal(t, string(content), testCase.expectedContent, "Unexpected content in devspace.yaml in testCase %s", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "No or wrong error in testCase %s", testCase.name)
	}
}

/*	configString := `version: v1beta2
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

	generatedConfig := &generated.Config{
		ActiveProfile: "dev-service1",
	}
	generated.SetTestConfig(generatedConfig)

	getConfigOnce = sync.Once{}
	GetConfig(context.Background(), "")

	err = SaveLoadedConfig()
	assert.NilError(t, err, "Error saving loaded config")
	configContent, err := fsutil.ReadFile(constants.DefaultConfigPath, -1)
	assert.NilError(t, err, "Error reading config file after save. Maybe it was not saved")
	expectedContent := `version: v1beta2
images:
  default:
    image: defaultImage
deployments:
- name: default
  helm:
    chart:
      name: ./chart
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
  selectors:
  - name: someSelector
    labelSelector: null
hooks:
- command: echo
cluster:
  kubeContext: someKubeContext
  namespace: LoadedNS
`
	assert.Equal(t, expectedContent, string(configContent), "Config differently saved than loaded")
}
*/
