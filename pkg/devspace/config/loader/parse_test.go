package loader

import (
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/fsutil"
	fakekubeconfig "github.com/loft-sh/devspace/pkg/util/kubeconfig/testing"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/ptr"
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
	profileObjects, err := loader.GetProfiles()
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
				options: &ConfigOptions{
					GeneratedConfig: testCase.generatedConfig,
					ConfigPath:      f.Name(),
				},
				kubeConfigLoader: &fakekubeconfig.Loader{
					RawConfig: &api.Config{},
				},
			}

			commands, err := loader.ParseCommands()
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
- name: test2
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
					},
					{
						Name: "test2",
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
		newConfig, err := NewConfigLoader(testCase.in.options, log.Discard).(*configLoader).parseConfig(testMap)
		if testCase.expectedErr {
			if err == nil {
				t.Fatalf("TestCase %s: expected error, but got none", index)
			} else {
				continue
			}
		} else if err != nil {
			t.Fatal(err)
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
