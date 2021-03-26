package loader

import (
	config2 "github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	fakegenerated "github.com/loft-sh/devspace/pkg/devspace/config/generated/testing"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/fsutil"
	fakekubeconfig "github.com/loft-sh/devspace/pkg/util/kubeconfig/testing"
	"github.com/loft-sh/devspace/pkg/util/log"
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
		configPath: testCase.configPath,
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

	configPath        string
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
			name:       "Get from custom config file with profile",
			configPath: "custom.yaml",
			options:    ConfigOptions{},
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
			name:    "Get from default file without profile",
			options: ConfigOptions{},
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
		configPath: testCase.configPath,
		kubeConfigLoader: &fakekubeconfig.Loader{
			RawConfig: &api.Config{},
		},
	}

	var config config2.Config
	var err error
	testCase.options.generatedLoader = &fakegenerated.Loader{
		Config: testCase.returnedGenerated,
	}
	config, err = loader.Load(&testCase.options, log.Discard)
	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Error in testCase %s", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
	}

	configAsYaml, err := yaml.Marshal(config.Config())
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

	expectedExists     bool
	expectedWorkDir    string
	expectedConfigPath string
	expectedErr        string
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
			expectedExists:     false,
			expectedWorkDir:    dir,
			expectedConfigPath: "custom.yaml",
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
		setDevSpaceRootTestCase{
			name:       "Custom config in subdir exists",
			configPath: "subdir/custom.yaml",
			files: map[string]interface{}{
				"subdir/custom.yaml": "",
			},
			expectedExists:     true,
			expectedWorkDir:    filepath.Join(dir, "subdir"),
			expectedConfigPath: "custom.yaml",
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
		configPath: testCase.configPath,
	}

	exists, err := loader.SetDevSpaceRoot(log.Discard)

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Error in testCase %s", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
	}
	assert.Equal(t, exists, testCase.expectedExists, "Unexpected existence answer in testCase %s", testCase.name)

	wd, err := os.Getwd()
	assert.NilError(t, err, "Error getting wd in testCase %s", testCase.name)
	assert.Equal(t, wd, testCase.expectedWorkDir, "Unexpected work dir in testCase %s", testCase.name)

	assert.Equal(t, loader.configPath, testCase.expectedConfigPath, "Unexpected configPath in testCase %s", testCase.name)
}

type getProfilesTestCase struct {
	name string

	files      map[string]interface{}
	configPath string

	expectedProfiles []string
	expectedErr      string
}

func TestGetProfiles(t *testing.T) {
	testCases := []getProfilesTestCase{
		getProfilesTestCase{
			name: "Empty file",
			files: map[string]interface{}{
				"devspace.yaml": map[interface{}]interface{}{
					"version": "v1beta9",
				},
			},
		},
		getProfilesTestCase{
			name:       "Parse several profiles",
			configPath: "custom.yaml",
			files: map[string]interface{}{
				"custom.yaml": map[interface{}]interface{}{
					"version": "v1beta9",
					"profiles": []interface{}{
						map[interface{}]interface{}{
							"name": "myprofile",
						},
					},
				},
			},
			expectedProfiles: []string{"myprofile"},
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
		testGetProfiles(testCase, t)
	}
}

func testGetProfiles(testCase getProfilesTestCase, t *testing.T) {
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
		configPath: testCase.configPath,
	}
	c, err := loader.Load(nil, log.Discard)
	assert.NilError(t, err, "Error loading config in testCase %s", testCase.name)
	profileObjects, err := c.Profiles()
	assert.NilError(t, err, "Error loading profiles in testCase %s", testCase.name)

	profiles := []string{}
	for _, p := range profileObjects {
		profiles = append(profiles, p.Name)
	}

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Error in testCase %s", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
	}

	assert.Equal(t, strings.Join(profiles, ","), strings.Join(testCase.expectedProfiles, ","), "Unexpected profiles in testCase %s", testCase.name)
}

type parseCommandsTestCase struct {
	name string

	generatedConfig *generated.Config
	data            map[interface{}]interface{}

	expectedCommands []*latest.CommandConfig
	expectedErr      string
}

// TODO: Finish this test!
func TestParseCommands(t *testing.T) {
	testCases := []parseCommandsTestCase{
		parseCommandsTestCase{
			data: map[interface{}]interface{}{
				"version": latest.Version,
			},
		},
	}

	for idx, testCase := range testCases {
		t.Run("Test "+strconv.Itoa(idx), func(t *testing.T) {
			f, err := ioutil.TempFile("", "")
			if err != nil {
				t.Fatal(err)
			}

			defer os.Remove(f.Name())

			out, err := yaml.Marshal(testCase.data)
			if err != nil {
				t.Fatal(err)
			}

			_, err = f.Write(out)
			if err != nil {
				t.Fatal(err)
			}

			// Close before reading
			f.Close()
			loader := &configLoader{
				configPath: f.Name(),
				kubeConfigLoader: &fakekubeconfig.Loader{
					RawConfig: &api.Config{},
				},
			}

			commandsInterface, err := loader.LoadWithParser(NewCommandsParser(), &ConfigOptions{
				GeneratedConfig: testCase.generatedConfig,
			}, log.Discard)
			if testCase.expectedErr == "" {
				assert.NilError(t, err, "Error in testCase %s", testCase.name)
			} else {
				assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
			}
			commands := commandsInterface.Config().Commands

			commandsAsYaml, err := yaml.Marshal(commands)
			assert.NilError(t, err, "Error parsing commands in testCase %s", testCase.name)
			expectedAsYaml, err := yaml.Marshal(testCase.expectedCommands)
			assert.NilError(t, err, "Error parsing expection to yaml in testCase %s", testCase.name)
			assert.Equal(t, string(commandsAsYaml), string(expectedAsYaml), "Unexpected commands in testCase %s", testCase.name)
		})
	}
}

type parseTestCase struct {
	in          *parseTestCaseInput
	expected    *latest.Config
	expectedErr bool
}

type parseTestCaseInput struct {
	config          string
	options         *ConfigOptions
	generatedConfig *generated.Config
}

func TestParseConfig(t *testing.T) {
	testCases := map[string]*parseTestCase{
		"Simple": {
			in: &parseTestCaseInput{
				config: `
version: v1alpha1`,
				options:         &ConfigOptions{},
				generatedConfig: &generated.Config{Vars: map[string]string{}},
			},
			expected: &latest.Config{
				Version: latest.Version,
				Dev: &latest.DevConfig{
					Interactive: &latest.InteractiveConfig{
						DefaultEnabled: ptr.Bool(true),
					},
				},
			},
		},
		"Simple with deployments": {
			in: &parseTestCaseInput{
				config: `
version: v1beta3
deployments:
- name: test
  component:
    containers:
    - image: nginx`,
				options:         &ConfigOptions{},
				generatedConfig: &generated.Config{Vars: map[string]string{}},
			},
			expected: &latest.Config{
				Version: latest.Version,
				Dev:     &latest.DevConfig{},
				Deployments: []*latest.DeploymentConfig{
					{
						Name: "test",
						Helm: &latest.HelmConfig{
							ComponentChart: ptr.Bool(true),
							V2:             true,
							Values: map[interface{}]interface{}{
								"containers": []interface{}{
									map[interface{}]interface{}{
										"image": "nginx",
									},
								},
							},
						},
					},
				},
			},
		},
		"Variables": {
			in: &parseTestCaseInput{
				config: `
version: v1beta3
deployments:
- name: ${my_var}
  component:
    containers:
    - image: nginx`,
				options: &ConfigOptions{},
				generatedConfig: &generated.Config{Vars: map[string]string{
					"my_var": "test",
				}},
			},
			expected: &latest.Config{
				Version: latest.Version,
				Dev:     &latest.DevConfig{},
				Deployments: []*latest.DeploymentConfig{
					{
						Name: "test",
						Helm: &latest.HelmConfig{
							ComponentChart: ptr.Bool(true),
							V2:             true,
							Values: map[interface{}]interface{}{
								"containers": []interface{}{
									map[interface{}]interface{}{
										"image": "nginx",
									},
								},
							},
						},
					},
				},
			},
		},
		"Profile replace with variable": {
			in: &parseTestCaseInput{
				config: `
version: v1beta3
deployments:
- name: ${does-not-exist}
  component:
    containers:
    - image: nginx
profiles:
- name: testprofile
  replace:
    deployments:
    - name: ${test_var}
			component:
				containers:
				- image: ubuntu`,
				options: &ConfigOptions{Profile: "testprofile"},
				generatedConfig: &generated.Config{Vars: map[string]string{
					"test_var": "test",
				}},
			},
			expected: &latest.Config{
				Version: latest.Version,
				Dev:     &latest.DevConfig{},
				Deployments: []*latest.DeploymentConfig{
					{
						Name: "test",
						Helm: &latest.HelmConfig{
							ComponentChart: ptr.Bool(true),
							V2:             true,
							Values: map[interface{}]interface{}{
								"containers": []interface{}{
									map[interface{}]interface{}{
										"image": "ubuntu",
									},
								},
							},
						},
					},
				},
			},
		},
		"Profile with patches": {
			in: &parseTestCaseInput{
				config: `
version: v1beta3
deployments:
- name: ${does-not-exist}
  component:
    containers:
		- image: nginx
commands:
- name: test
	command: should not show up
vars:
- name: unused
	default: unused
profiles:
- name: testprofile
	patches:
	- op: replace
		path: deployments[0].component.containers[0].image
		value: ${test_var}
	- op: replace
		path: deployments[0].name
		value: ${test_var_2}`,
				options: &ConfigOptions{Profile: "testprofile", Vars: []string{"test_var=ubuntu"}},
				generatedConfig: &generated.Config{Vars: map[string]string{
					"test_var_2": "test",
				}},
			},
			expected: &latest.Config{
				Version: latest.Version,
				Dev:     &latest.DevConfig{},
				Deployments: []*latest.DeploymentConfig{
					{
						Name: "test",
						Helm: &latest.HelmConfig{
							ComponentChart: ptr.Bool(true),
							V2:             true,
							Values: map[interface{}]interface{}{
								"containers": []interface{}{
									map[interface{}]interface{}{
										"image": "ubuntu",
									},
								},
							},
						},
					},
				},
			},
		},
		"Commands": {
			in: &parseTestCaseInput{
				config: `
version: v1beta3
deployments:
- name: ${test_var}
  component:
    containers:
		- image: nginx
commands:
- name: test
	command: should not show up
vars:
- name: unused
	default: unused
profiles:
- name: testprofile
	patches:
	- op: replace
		path: deployments[0].component.containers[0].image
		value: ${unused}
	- op: replace
		path: deployments[0].name
		value: ${should-not-show-up}`,
				options:         &ConfigOptions{Vars: []string{"test_var=test"}},
				generatedConfig: &generated.Config{Vars: map[string]string{}},
			},
			expected: &latest.Config{
				Version: latest.Version,
				Dev:     &latest.DevConfig{},
				Deployments: []*latest.DeploymentConfig{
					{
						Name: "test",
						Helm: &latest.HelmConfig{
							ComponentChart: ptr.Bool(true),
							V2:             true,
							Values: map[interface{}]interface{}{
								"containers": []interface{}{
									map[interface{}]interface{}{
										"image": "nginx",
									},
								},
							},
						},
					},
				},
			},
		},
		"Default variables": {
			in: &parseTestCaseInput{
				config: `
version: v1beta6
deployments:
- name: ${new}
	helm:
		componentChart: true
		values:
		  containers:
		  - image: nginx
commands:
- name: test
	command: should not show up
vars:
- name: abc
	default: test
profiles:
- name: testprofile
	patches:
	- op: replace
		path: vars[0].name
		value: new`,
				options:         &ConfigOptions{Profile: "testprofile"},
				generatedConfig: &generated.Config{Vars: map[string]string{"new": "newdefault"}},
			},
			expected: &latest.Config{
				Version: latest.Version,
				Dev:     &latest.DevConfig{},
				Deployments: []*latest.DeploymentConfig{
					{
						Name: "newdefault",
						Helm: &latest.HelmConfig{
							ComponentChart: ptr.Bool(true),
							Values: map[interface{}]interface{}{
								"containers": []interface{}{
									map[interface{}]interface{}{
										"image": "nginx",
									},
								},
							},
						},
					},
				},
			},
		},
		"Variable source none": {
			in: &parseTestCaseInput{
				config: `
version: v1beta6
deployments:
- name: ${new}
  kubectl:
    manifests:
    - test.yaml
vars:
- name: new
  source: none
  default: test`,
				options:         &ConfigOptions{},
				generatedConfig: &generated.Config{Vars: map[string]string{}},
			},
			expected: &latest.Config{
				Version: latest.Version,
				Dev:     &latest.DevConfig{},
				Deployments: []*latest.DeploymentConfig{
					{
						Name: "test",
						Kubectl: &latest.KubectlConfig{
							Manifests: []string{
								"test.yaml",
							},
						},
					},
				},
			},
		},
		"Profile parent": {
			in: &parseTestCaseInput{
				config: `
version: v1beta7
deployments:
- name: test
  kubectl: 
    manifests:
    - test.yaml
- name: test2
  kubectl: 
    manifests:
    - test.yaml
profiles:
- name: parent
	replace: 
		images:
			test:
				image: test
- name: beforeParent
	parent: parent
	patches:
	- op: replace
		path: deployments[0].name
		value: replaced
- name: test
	parent: beforeParent
	patches:
	- op: replace
		path: deployments[0].name
		value: replaced2`,
				options:         &ConfigOptions{Profile: "test"},
				generatedConfig: &generated.Config{Vars: map[string]string{}},
			},
			expected: &latest.Config{
				Version: latest.Version,
				Dev:     &latest.DevConfig{},
				Deployments: []*latest.DeploymentConfig{
					{
						Name: "replaced2",
						Kubectl: &latest.KubectlConfig{
							Manifests: []string{
								"test.yaml",
							},
						},
					},
					{
						Name: "test2",
						Kubectl: &latest.KubectlConfig{
							Manifests: []string{
								"test.yaml",
							},
						},
					},
				},
				Images: map[string]*latest.ImageConfig{
					"test": &latest.ImageConfig{
						Image:                 "test",
						PreferSyncOverRebuild: true,
					},
				},
			},
		},
		"Profile loop error": {
			in: &parseTestCaseInput{
				config: `
version: v1beta7
deployments:
- name: test
- name: test2
profiles:
- name: parent
	parent: test
	replace: 
		images:
			test:
				image: test
- name: beforeParent
	parent: parent
	patches:
	- op: replace
		path: deployments[0].name
		value: replaced
- name: test
	parent: beforeParent
	patches:
	- op: replace
		path: deployments[1].name
		value: replaced2`,
				options:         &ConfigOptions{Profile: "test"},
				generatedConfig: &generated.Config{Vars: map[string]string{}},
			},
			expectedErr: true,
		},
		"Profile strategic merge": {
			in: &parseTestCaseInput{
				config: `
version: v1beta9
images:
  test: 
    image: test/test
  delete: 
    image: test/test
deployments:
- name: test
  helm:
    values:
      service:
        ports:
        - port: 3000
      containers:
      - image: test/test
      - image: test456/test456
- name: test2
  helm:
    componentChart: true
    values:
      containers:
      - image: test/test
profiles:
- name: test
  strategicMerge:
    images:
      test:
        image: test2/test2
      delete: null
    deployments:
    - name: test
      helm:
        componentChart: true
        values:
          containers:
          - image: test123/test123`,
				options:         &ConfigOptions{Profile: "test"},
				generatedConfig: &generated.Config{Vars: map[string]string{}},
			},
			expected: &latest.Config{
				Version: latest.Version,
				Dev:     &latest.DevConfig{},
				Images: map[string]*latest.ImageConfig{
					"test": {
						Image: "test2/test2",
					},
				},
				Deployments: []*latest.DeploymentConfig{
					{
						Name: "test",
						Helm: &latest.HelmConfig{
							ComponentChart: ptr.Bool(true),
							Values: map[interface{}]interface{}{
								"service": map[interface{}]interface{}{
									"ports": []interface{}{
										map[interface{}]interface{}{
											"port": 3000,
										},
									},
								},
								"containers": []interface{}{
									map[interface{}]interface{}{
										"image": "test123/test123",
									},
								},
							},
						},
					},
					{
						Name: "test2",
						Helm: &latest.HelmConfig{
							ComponentChart: ptr.Bool(true),
							Values: map[interface{}]interface{}{
								"containers": []interface{}{
									map[interface{}]interface{}{
										"image": "test/test",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Execute test cases
	for index, testCase := range testCases {
		testMap := map[interface{}]interface{}{}
		err := yaml.Unmarshal([]byte(strings.Replace(testCase.in.config, "	", "  ", -1)), &testMap)
		if err != nil {
			t.Fatal(err)
		}

		testCase.in.options.GeneratedConfig = testCase.in.generatedConfig
		testCase.in.options.generatedLoader = &fakeGeneratedLoader{}

		configLoader := NewConfigLoader("").(*configLoader)
		newConfig, _, _, err := configLoader.parseConfig(testMap, NewDefaultParser(), testCase.in.options, log.Discard)
		if testCase.expectedErr {
			if err == nil {
				t.Fatalf("TestCase %s: expected error, but got none", index)
			} else {
				continue
			}
		} else if err != nil {
			t.Fatalf("Error %v in case %s", err, index)
		}

		if !reflect.DeepEqual(newConfig, testCase.expected) {
			newConfigYaml, _ := yaml.Marshal(newConfig)
			expectedYaml, _ := yaml.Marshal(testCase.expected)
			if string(newConfigYaml) != string(expectedYaml) {
				t.Fatalf("TestCase %s: Got %s, but expected %s", index, newConfigYaml, expectedYaml)
			}
		}
	}
}

type fakeGeneratedLoader struct{}

func (f *fakeGeneratedLoader) Load() (*generated.Config, error) {
	panic("unimplemented")
}
func (f *fakeGeneratedLoader) LoadFromPath(path string) (*generated.Config, error) {
	panic("unimplemented")
}
func (f *fakeGeneratedLoader) Save(config *generated.Config) error {
	return nil
}
