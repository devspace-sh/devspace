package generated

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/loft-sh/devspace/pkg/util/fsutil"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	yaml "gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type loadTestCase struct {
	name string

	profile string
	files   map[string]interface{}

	expectedConfig *Config
	expectedErr    string
}

func TestLoad(t *testing.T) {
	testCases := []loadTestCase{
		loadTestCase{
			name:           "no config file",
			expectedConfig: &Config{},
		},
		loadTestCase{
			name: "load empty config file",
			files: map[string]interface{}{
				".devspace/generated.yaml": struct{}{},
			},
			profile: "someprofile",
			expectedConfig: &Config{
				OverrideProfile: ptr.String("someprofile"),
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

	for _, testCase := range testCases {
		testLoad(testCase, t)
	}
}

func testLoad(testCase loadTestCase, t *testing.T) {
	defer func() {
		for _, path := range []string{".devspace/generated.yaml"} {
			os.Remove(path)
		}
	}()
	for path, data := range testCase.files {
		dataAsYaml, err := yaml.Marshal(data)
		assert.NilError(t, err, "Error parsing data of file %s in testCase %s", path, testCase.name)
		err = fsutil.WriteToFile([]byte(dataAsYaml), path)
		assert.NilError(t, err, "Error writing file %s in testCase %s", path, testCase.name)
	}

	loader := NewConfigLoader(testCase.profile)

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

type saveTestCase struct {
	name               string
	devspaceConfigPath string

	config *Config

	expectedConfigFileName string
	expectedConfigFile     interface{}
	expectedErr            string
}

func TestSave(t *testing.T) {
	testCases := []saveTestCase{
		saveTestCase{
			name: "Save config default",
			config: &Config{
				OverrideProfile: ptr.String("overrideProf"),
				ActiveProfile:   "active",
				Vars: map[string]string{
					"key": "value",
				},
			},
			expectedConfigFileName: ".devspace/generated.yaml",
			expectedConfigFile: Config{
				OverrideProfile: ptr.String("overrideProf"),
				ActiveProfile:   "active",
				Vars: map[string]string{
					"key": "value",
				},
			},
		},
		saveTestCase{
			name:               "Save config test.yaml",
			devspaceConfigPath: "test.yaml",
			config: &Config{
				OverrideProfile: ptr.String("overrideProf"),
				ActiveProfile:   "active",
				Vars: map[string]string{
					"key": "value",
				},
			},
			expectedConfigFileName: ".devspace/generated-test.yaml",
			expectedConfigFile: Config{
				OverrideProfile: ptr.String("overrideProf"),
				ActiveProfile:   "active",
				Vars: map[string]string{
					"key": "value",
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

	for _, testCase := range testCases {
		var loader ConfigLoader
		if testCase.devspaceConfigPath != "" {
			loader = NewConfigLoaderFromDevSpacePath("", testCase.devspaceConfigPath)
		} else {
			loader = NewConfigLoader("")
		}

		err := loader.Save(testCase.config)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		fileContent, err := ioutil.ReadFile(testCase.expectedConfigFileName)
		assert.NilError(t, err, "Error reading file in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedConfigFile)
		assert.NilError(t, err, "Error parsing expection to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(fileContent), string(expectedAsYaml), "Unexpected config file in testCase %s", testCase.name)
	}
}

type getActiveTestCase struct {
	name string

	active   string
	override *string
	profiles map[string]*CacheConfig

	expectedCache CacheConfig
}

func TestGetActive(t *testing.T) {
	testCases := []getActiveTestCase{
		getActiveTestCase{
			name:     "Get overriden profile that is not there",
			active:   "acttive",
			override: ptr.String("override"),
			profiles: map[string]*CacheConfig{},

			expectedCache: CacheConfig{},
		},
	}

	for _, testCase := range testCases {
		config := &Config{
			ActiveProfile:   testCase.active,
			OverrideProfile: testCase.override,
			Profiles:        testCase.profiles,
		}

		active := config.GetActive()

		activeAsYaml, err := yaml.Marshal(active)
		assert.NilError(t, err, "Error parsing active provider in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedCache)
		assert.NilError(t, err, "Error parsing expection to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(activeAsYaml), string(expectedAsYaml), "Unexpected config file in testCase %s", testCase.name)
	}
}

func TestGetCaches(t *testing.T) {
	dsConfig := &Config{
		Profiles: map[string]*CacheConfig{
			"SomeConfig": &CacheConfig{},
		},
	}
	InitDevSpaceConfig(dsConfig, "SomeConfig")
	cacheConfig := dsConfig.Profiles["SomeConfig"]
	assert.Equal(t, 0, len(cacheConfig.Deployments), "Deployments wrong initialized")
	assert.Equal(t, 0, len(cacheConfig.Images), "Images wrong initialized")
	assert.Equal(t, 0, len(cacheConfig.Dependencies), "Dependencies wrong initialized")

	imageCache := cacheConfig.GetImageCache("NewImageCache")
	assert.Equal(t, 1, len(cacheConfig.Images), "New imageCache not added to cache")
	assert.Equal(t, "", imageCache.ImageConfigHash, "ImageCache wrong initialized")
	assert.Equal(t, "", imageCache.DockerfileHash, "ImageCache wrong initialized")
	assert.Equal(t, "", imageCache.ContextHash, "ImageCache wrong initialized")
	assert.Equal(t, "", imageCache.EntrypointHash, "ImageCache wrong initialized")
	assert.Equal(t, "", imageCache.CustomFilesHash, "ImageCache wrong initialized")
	assert.Equal(t, "", imageCache.ImageName, "ImageCache wrong initialized")
	assert.Equal(t, "", imageCache.Tag, "ImageCache wrong initialized")

	deploymentCache := cacheConfig.GetDeploymentCache("NewDeploymentCache")
	assert.Equal(t, 1, len(cacheConfig.Deployments), "New deploymentCache not added to cache")
	assert.Equal(t, "", deploymentCache.DeploymentConfigHash, "DeploymentCache wrong initialized")
	assert.Equal(t, "", deploymentCache.HelmOverridesHash, "DeploymentCache wrong initialized")
	assert.Equal(t, "", deploymentCache.HelmChartHash, "DeploymentCache wrong initialized")
	assert.Equal(t, "", deploymentCache.KubectlManifestsHash, "DeploymentCache wrong initialized")
}
