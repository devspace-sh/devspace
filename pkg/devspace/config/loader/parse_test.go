package loader

import (
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	fakekubeconfig "github.com/devspace-cloud/devspace/pkg/util/kubeconfig/testing"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	yaml "gopkg.in/yaml.v2"
	"gotest.tools/assert"
	"k8s.io/client-go/tools/clientcmd/api"
)

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
				"devspace.yaml": map[interface{}]interface{}{},
			},
		},
		getProfilesTestCase{
			name:       "Parse several profiles",
			configPath: "custom.yaml",
			files: map[string]interface{}{
				"custom.yaml": map[interface{}]interface{}{
					"profiles": []interface{}{
						"noMap",
						map[interface{}]interface{}{
							"description": "Has no name",
						},
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
		options: &ConfigOptions{
			ConfigPath: testCase.configPath,
		},
	}
	profiles, err := loader.GetProfiles()

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Error in testCase %s", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
	}

	assert.Equal(t, strings.Join(profiles, ", "), strings.Join(testCase.expectedProfiles, ", "), "Unexpected profiles in testCase %s", testCase.name)
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

	for _, testCase := range testCases {
		loader := &configLoader{
			options: &ConfigOptions{},
			kubeConfigLoader: &fakekubeconfig.Loader{
				RawConfig: &api.Config{},
			},
		}

		commands, err := loader.ParseCommands(testCase.generatedConfig, testCase.data)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		commandsAsYaml, err := yaml.Marshal(commands)
		assert.NilError(t, err, "Error parsing commands in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedCommands)
		assert.NilError(t, err, "Error parsing expection to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(commandsAsYaml), string(expectedAsYaml), "Unexpected commands in testCase %s", testCase.name)
	}
}

type parseTestCase struct {
	in       *parseTestCaseInput
	expected *latest.Config
}

type parseTestCaseInput struct {
	config          string
	options         *ConfigOptions
	generatedConfig *generated.Config
}

func TestParseConfig(t *testing.T) {
	testCases := []*parseTestCase{
		{
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
		{
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
		{
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
		{
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
		{
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
		{
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
	}

	// Execute test cases
	for index, testCase := range testCases {
		testMap := map[interface{}]interface{}{}
		err := yaml.Unmarshal([]byte(strings.Replace(testCase.in.config, "	", "  ", -1)), &testMap)
		if err != nil {
			t.Fatal(err)
		}

		newConfig, err := NewConfigLoader(testCase.in.options, log.Discard).(*configLoader).parseConfig(testCase.in.generatedConfig, testMap)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(newConfig, testCase.expected) {
			newConfigYaml, _ := yaml.Marshal(newConfig)
			expectedYaml, _ := yaml.Marshal(testCase.expected)

			t.Fatalf("TestCase %d: Got %s, but expected %s", index, newConfigYaml, expectedYaml)
		}
	}
}
