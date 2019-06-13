package configutil

import (
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"

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
	GetConfig()
	if config == nil {
		t.Fatal("Config is nil after initializing")
	}
	assert.Equal(t, *config.Version, latest.Version, "Initialized config has wrong version")
	assert.Equal(t, *config.Cluster.KubeContext, "someKubeContext", "Initialized config has wrong kubeContext of cluster")
	assert.Equal(t, *config.Cluster.Namespace, "someNS", "Initialized config has wrong namespace of cluster")
	assert.Equal(t, len(*config.Images), 1, "Initialized config has wrong number of images")
	assert.Equal(t, *(*config.Images)["default"].Image, "defaultImage", "Initialized config has wrong image")
}
