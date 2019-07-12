package configutil

import (
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"gotest.tools/assert"
)

func TestRestoreVars(t *testing.T) {
	testConfig := latest.NewRaw()
	testConfig.Deployments = &[]*latest.DeploymentConfig{}
	testConfig.Cluster = &latest.Cluster{
		Namespace: ptr.String("UnloadedNS"),
	}

	LoadedVars[".cluster.namespace"] = "LoadedNS"

	resultConfig, err := RestoreVars(testConfig)

	assert.NilError(t, err, "Error Restoring Vars")
	assert.Equal(t, "LoadedNS", *resultConfig.Cluster.Namespace, "Loaded var not correctly applied")
}

func TestSaveLoadedConfig(t *testing.T) {
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
		ActiveConfig: "dev-service1",
	}
	generated.SetTestConfig(generatedConfig)

	getConfigOnce = sync.Once{}
	GetConfigWithoutDefaults(true)

	/*err = os.Remove(constants.DefaultConfigPath)
	assert.NilError(t, err, "Error removing config files")
	err = os.Remove(constants.DefaultConfigsPath)
	assert.NilError(t, err, "Error removing config files")*/

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
