package localcache

import (
	"os"
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/util/fsutil"
	yaml "gopkg.in/yaml.v3"
	"gotest.tools/assert"
)

type loadTestCase struct {
	name string

	files map[string]interface{}

	expectedConfig *LocalCache
	expectedErr    string
}

func TestLoad(t *testing.T) {
	testCases := []loadTestCase{
		{
			name:           "no config file",
			expectedConfig: &LocalCache{},
		},
		{
			name: "load empty config file",
			files: map[string]interface{}{
				".devspace/generated.yaml": struct{}{},
			},
			expectedConfig: &LocalCache{},
		},
	}

	dir := t.TempDir()

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
	}()

	for _, testCase := range testCases {
		testLoad(testCase, t)
	}
}

func testLoad(testCase loadTestCase, t *testing.T) {
	defer func() {
		for _, path := range []string{".devspace/cache.yaml"} {
			os.Remove(path)
		}
	}()
	for path, data := range testCase.files {
		dataAsYaml, err := yaml.Marshal(data)
		assert.NilError(t, err, "Error parsing data of file %s in testCase %s", path, testCase.name)
		err = fsutil.WriteToFile([]byte(dataAsYaml), path)
		assert.NilError(t, err, "Error writing file %s in testCase %s", path, testCase.name)
	}

	loader := NewCacheLoader()

	config, err := loader.Load(constants.DefaultConfigPath)

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Error in testCase %s", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
	}

	configAsYaml, err := yaml.Marshal(config)
	assert.NilError(t, err, "Error parsing config in testCase %s", testCase.name)
	expectedAsYaml, err := yaml.Marshal(testCase.expectedConfig)
	assert.NilError(t, err, "Error parsing exception to yaml in testCase %s", testCase.name)
	assert.Equal(t, string(configAsYaml), string(expectedAsYaml), "Unexpected config in testCase %s", testCase.name)
}

type saveTestCase struct {
	name string

	config *LocalCache

	expectedConfigFileName string
	expectedConfigFile     interface{}
	expectedErr            string
}

func TestSave(t *testing.T) {
	testCases := []saveTestCase{
		{
			name: "Save config default",
			config: &LocalCache{
				Vars: map[string]string{
					"key": "value",
				},
			},
			expectedConfigFileName: ".devspace/cache.yaml",
			expectedConfigFile: LocalCache{
				Vars: map[string]string{
					"key": "value",
				},
			},
		},
		{
			name: "Save config test.yaml",
			config: &LocalCache{
				Vars: map[string]string{
					"key": "value",
				},
			},
			expectedConfigFileName: ".devspace/cache-test.yaml",
			expectedConfigFile: LocalCache{
				Vars: map[string]string{
					"key": "value",
				},
			},
		},
	}

	dir := t.TempDir()

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
	}()

	for _, testCase := range testCases {
		testCase.config.cachePath = testCase.expectedConfigFileName
		err := testCase.config.Save()

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		fileContent, err := os.ReadFile(testCase.expectedConfigFileName)
		assert.NilError(t, err, "Error reading file in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedConfigFile)
		assert.NilError(t, err, "Error parsing exception to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(fileContent), string(expectedAsYaml), "Unexpected config file in testCase %s", testCase.name)
	}
}
