package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/legacy"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	homedir "github.com/mitchellh/go-homedir"
	yaml "gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type getProviderTestCase struct {
	name string

	providers []*latest.Provider
	needle    string

	expectedProvider *latest.Provider
}

func TestGetProvider(t *testing.T) {
	testCases := []getProviderTestCase{
		getProviderTestCase{
			name: "Found",
			providers: []*latest.Provider{
				&latest.Provider{
					Name: "NotThis",
				},
				&latest.Provider{
					Name: "thisOne",
				},
			},
			needle: "thisOne",
			expectedProvider: &latest.Provider{
				Name: "thisOne",
			},
		},
		getProviderTestCase{
			name: "Not found",
			providers: []*latest.Provider{
				&latest.Provider{
					Name: "NotThis",
				},
			},
			needle: "notThere",
		},
	}

	for _, testCase := range testCases {
		config := &latest.Config{
			Providers: testCase.providers,
		}
		provider := GetProvider(config, testCase.needle)

		providerAsYaml, err := yaml.Marshal(provider)
		assert.NilError(t, err, "Error parsing provider to yaml in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedProvider)
		assert.NilError(t, err, "Error parsing expection to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(providerAsYaml), string(expectedAsYaml), "Unexpected provider in testCase %s", testCase.name)
	}
}

type saveTestCase struct {
	name string

	config *latest.Config

	expectedConfigFile interface{}
	expectedErr        string
}

func TestSave(t *testing.T) {
	testCases := []saveTestCase{
		saveTestCase{
			name: "Save config",
			config: &latest.Config{
				Version: "latest",
				Default: "myDefault",
			},
			expectedConfigFile: latest.Config{
				Version: "latest",
				Default: "myDefault",
			},
		},
	}

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

	defer func() {
		//Delete temp folder
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}()

	homedir, err := homedir.Dir()
	assert.NilError(t, err, "Error getting Homedir")
	DevSpaceProvidersConfigPath, err = filepath.Rel(homedir, filepath.Join(dir, "providers.yaml"))
	assert.NilError(t, err, "Error setting config path")

	for _, testCase := range testCases {
		loader := NewLoader()

		err := loader.Save(testCase.config)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		fileContent, err := ioutil.ReadFile("providers.yaml")
		assert.NilError(t, err, "Error reading file in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedConfigFile)
		assert.NilError(t, err, "Error parsing expection to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(fileContent), string(expectedAsYaml), "Unexpected config file in testCase %s", testCase.name)
	}
}

type loadTestCase struct {
	name string

	files map[string]interface{}

	expectedConfig *latest.Config
	expectedErr    string
}

func TestLoad(t *testing.T) {
	testCases := []loadTestCase{
		loadTestCase{
			name: "No config files",
			expectedConfig: &latest.Config{
				Version: latest.Version,
				Providers: []*latest.Provider{
					DevSpaceCloudProviderConfig,
				},
			},
		},
		loadTestCase{
			name: "Get latest config and add devspace-cloud provider",
			files: map[string]interface{}{
				"providers.yaml": latest.Config{},
			},
			expectedConfig: &latest.Config{
				Version: latest.Version,
				Providers: []*latest.Provider{
					&latest.Provider{
						Name: DevSpaceCloudProviderName,
						Host: "https://app.devspace.cloud",
					},
				},
			},
		},
		loadTestCase{
			name: "Get latest config with devspace-cloud provider",
			files: map[string]interface{}{
				"providers.yaml": latest.Config{
					Providers: []*latest.Provider{
						&latest.Provider{
							Name: DevSpaceCloudProviderName,
						},
					},
				},
			},
			expectedConfig: &latest.Config{
				Version: latest.Version,
				Providers: []*latest.Provider{
					&latest.Provider{
						Name: DevSpaceCloudProviderName,
						Host: "https://app.devspace.cloud",
					},
				},
			},
		},
		loadTestCase{
			name: "Get legacy config and add devspace-cloud provider",
			files: map[string]interface{}{
				"legacy.yaml": legacy.Config{},
			},
			expectedConfig: &latest.Config{
				Version: latest.Version,
				Providers: []*latest.Provider{
					&latest.Provider{
						Name: DevSpaceCloudProviderName,
						Host: "https://app.devspace.cloud",
					},
				},
			},
		},
		loadTestCase{
			name: "Get legacy config with devspace-cloud provider",
			files: map[string]interface{}{
				"legacy.yaml": legacy.Config{
					DevSpaceCloudProviderName: &legacy.Provider{
						Key:   "legacyKey1",
						Token: "legacyToken1",
					},
				},
			},
			expectedConfig: &latest.Config{
				Version: latest.Version,
				Providers: []*latest.Provider{
					&latest.Provider{
						Name:  DevSpaceCloudProviderName,
						Host:  "https://app.devspace.cloud",
						Key:   "legacyKey1",
						Token: "legacyToken1",
					},
				},
			},
		},
	}

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

	defer func() {
		//Delete temp folder
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}()

	homedir, err := homedir.Dir()
	assert.NilError(t, err, "Error getting Homedir")
	DevSpaceProvidersConfigPath, err = filepath.Rel(homedir, filepath.Join(dir, "providers.yaml"))
	assert.NilError(t, err, "Error setting config path")
	LegacyDevSpaceCloudConfigPath, err = filepath.Rel(homedir, filepath.Join(dir, "legacy.yaml"))
	assert.NilError(t, err, "Error setting legacy config path")

	for _, testCase := range testCases {
		testLoad(testCase, t)
	}
}

func testLoad(testCase loadTestCase, t *testing.T) {
	defer func() {
		for _, path := range []string{"providers.yaml", "legacy.yaml"} {
			os.Remove(path)
		}
	}()
	for path, data := range testCase.files {
		dataAsYaml, err := yaml.Marshal(data)
		assert.NilError(t, err, "Error parsing data of file %s in testCase %s", path, testCase.name)
		err = fsutil.WriteToFile([]byte(dataAsYaml), path)
		assert.NilError(t, err, "Error writing file %s in testCase %s", path, testCase.name)
	}

	loader := NewLoader()

	config, err := loader.Load()

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Error in testCase %s", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
	}

	configAsYaml, err := yaml.Marshal(config)
	assert.NilError(t, err, "Error parsing config in testCase %s", testCase.name)
	expectedAsYaml, err := yaml.Marshal(testCase.expectedConfig)
	assert.NilError(t, err, "Error parsing expection to yaml in testCase %s", testCase.name)
	assert.Equal(t, string(configAsYaml), string(expectedAsYaml), "Unexpected config in testCase %s", testCase.name)
}

type getDefaultProviderNameTestCase struct {
	name string

	config *latest.Config

	expectedDefault string
}

func TestGetDefaultProviderName(t *testing.T) {
	testCases := []getDefaultProviderNameTestCase{
		getDefaultProviderNameTestCase{
			name: "Get set default",
			config: &latest.Config{
				Default: "myDefault",
			},
			expectedDefault: "myDefault",
		},
	}

	for _, testCase := range testCases {
		loader := &loader{
			loadedConfig: testCase.config,
		}
		defaultProvider, _ := loader.GetDefaultProviderName()

		assert.Equal(t, defaultProvider, testCase.expectedDefault, "Unexpected provider name in testCase %s", testCase.name)
	}
}
