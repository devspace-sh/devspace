package loader

import (
	"reflect"
	"strings"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	yaml "gopkg.in/yaml.v2"
)

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
