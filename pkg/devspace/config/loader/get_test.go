package loader

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	fakegenerated "github.com/devspace-cloud/devspace/pkg/devspace/config/generated/testing"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	fakekubeconfig "github.com/devspace-cloud/devspace/pkg/util/kubeconfig/testing"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	yaml "gopkg.in/yaml.v2"
	"gotest.tools/assert"
	"k8s.io/client-go/tools/clientcmd/api"
)

type existsTestCase struct {
	name string

	files      map[string]interface{}
	configPath string

	expectedanswer bool
}

func TestExists(t *testing.T) {
	testCases := []existsTestCase{
		existsTestCase{
			name:       "Only custom file name exists",
			configPath: "mypath.yaml",
			files: map[string]interface{}{
				"mypath.yaml": "",
			},
			expectedanswer: true,
		},
		existsTestCase{
			name: "Default file name does not exist",
			files: map[string]interface{}{
				"mypath.yaml": "",
			},
			expectedanswer: false,
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
		testExists(testCase, t)
	}
}

func testExists(testCase existsTestCase, t *testing.T) {
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

	loader := &configLoader{
		options: &ConfigOptions{
			ConfigPath: testCase.configPath,
		},
	}

	exists := loader.Exists()
	assert.Equal(t, exists, testCase.expectedanswer, "Unexpected answer in testCase %s", testCase.name)
}

type cloneTestCase struct {
	name string

	cloner ConfigOptions

	expectedClone *ConfigOptions
	expectedErr   string
}

func TestClone(t *testing.T) {
	testCases := []cloneTestCase{
		cloneTestCase{
			name: "Clone ConfigOptions",
			cloner: ConfigOptions{
				Profile:     "clonerProf",
				KubeContext: "clonerContext",
			},
			expectedClone: &ConfigOptions{
				Profile:     "clonerProf",
				KubeContext: "clonerContext",
			},
		},
	}

	for _, testCase := range testCases {
		clone, err := (&testCase.cloner).Clone()

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		configAsYaml, err := yaml.Marshal(clone)
		assert.NilError(t, err, "Error parsing clone in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedClone)
		assert.NilError(t, err, "Error parsing expection to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(configAsYaml), string(expectedAsYaml), "Unexpected clone in testCase %s", testCase.name)
	}
}

type loadTestCase struct {
	name string

	options           ConfigOptions
	returnedGenerated generated.Config
	files             map[string]interface{}
	withProfile       bool

	expectedConfig *latest.Config
	expectedErr    string
}

func TestLoad(t *testing.T) {
	testCases := []loadTestCase{
		loadTestCase{
			name: "Get from custom config file with profile",
			options: ConfigOptions{
				ConfigPath: "custom.yaml",
			},
			files: map[string]interface{}{
				"custom.yaml": latest.Config{
					Version: latest.Version,
					Profiles: []*latest.ProfileConfig{
						&latest.ProfileConfig{
							Name: "active",
						},
					},
				},
			},
			returnedGenerated: generated.Config{
				ActiveProfile: "active",
			},
			withProfile: true,
			expectedConfig: &latest.Config{
				Version: latest.Version,
				Dev:     &latest.DevConfig{},
			},
		},
		loadTestCase{
			name: "Get from default file without profile",
			options: ConfigOptions{
				Profile: "myProfile",
			},
			files: map[string]interface{}{
				"devspace.yaml": latest.Config{
					Version: latest.Version,
				},
			},
			expectedConfig: &latest.Config{
				Version: latest.Version,
				Dev:     &latest.DevConfig{},
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
		for _, path := range []string{".devspace/generated.yaml", "devspace.yaml", "custom.yaml"} {
			os.Remove(path)
		}
	}()
	for path, data := range testCase.files {
		dataAsYaml, err := yaml.Marshal(data)
		assert.NilError(t, err, "Error parsing data of file %s in testCase %s", path, testCase.name)
		err = fsutil.WriteToFile([]byte(dataAsYaml), path)
		assert.NilError(t, err, "Error writing file %s in testCase %s", path, testCase.name)
	}

	loader := &configLoader{
		options: &testCase.options,
		log:     log.Discard,
		generatedLoader: &fakegenerated.Loader{
			Config: testCase.returnedGenerated,
		},
		kubeConfigLoader: &fakekubeconfig.Loader{
			RawConfig: &api.Config{},
		},
	}

	var config *latest.Config
	var err error
	if testCase.withProfile {
		config, err = loader.Load()
	} else {
		config, err = loader.LoadWithoutProfile()
	}

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

type setDevSpaceRootTestCase struct {
	name string

	configPath string
	startDir   string
	files      map[string]interface{}

	expectedExists  bool
	expectedWorkDir string
	expectedErr     string
}

func TestSetDevSpaceRoot(t *testing.T) {
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

	testCases := []setDevSpaceRootTestCase{
		setDevSpaceRootTestCase{
			name:       "No custom.yaml",
			configPath: "custom.yaml",
			files: map[string]interface{}{
				"devspace.yaml": "",
			},
			expectedExists:  false,
			expectedWorkDir: dir,
		},
		setDevSpaceRootTestCase{
			name:            "No devspace.yaml",
			expectedExists:  false,
			expectedWorkDir: dir,
		},
		setDevSpaceRootTestCase{
			name: "Config exists",
			files: map[string]interface{}{
				"devspace.yaml": "",
			},
			startDir:        "subDir",
			expectedExists:  true,
			expectedWorkDir: dir,
		},
	}

	for _, testCase := range testCases {
		testSetDevSpaceRoot(testCase, t)
	}
}

func testSetDevSpaceRoot(testCase setDevSpaceRootTestCase, t *testing.T) {
	wdBackup, err := os.Getwd()
	assert.NilError(t, err, "Error getting current working directory")
	defer func() {
		os.Chdir(wdBackup)
		for _, path := range []string{"devspace.yaml", "custom.yaml"} {
			os.Remove(path)
		}
	}()
	for path, data := range testCase.files {
		dataAsYaml, err := yaml.Marshal(data)
		assert.NilError(t, err, "Error parsing data of file %s in testCase %s", path, testCase.name)
		err = fsutil.WriteToFile([]byte(dataAsYaml), path)
		assert.NilError(t, err, "Error writing file %s in testCase %s", path, testCase.name)
	}
	if testCase.startDir != "" {
		os.Mkdir(testCase.startDir, os.ModePerm)
		os.Chdir(testCase.startDir)
	}

	loader := &configLoader{
		options: &ConfigOptions{
			ConfigPath: testCase.configPath,
		},
		log: log.Discard,
	}

	exists, err := loader.SetDevSpaceRoot()

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Error in testCase %s", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
	}
	assert.Equal(t, exists, testCase.expectedExists, "Unexpected existence answer in testCase %s", testCase.name)

	wd, err := os.Getwd()
	assert.NilError(t, err, "Error getting wd in testCase %s", testCase.name)
	assert.Equal(t, wd, testCase.expectedWorkDir, "Unexpected work dir in testCase %s", testCase.name)

}
