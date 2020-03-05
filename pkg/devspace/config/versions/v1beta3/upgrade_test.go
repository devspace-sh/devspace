package v1beta3

import (
	"reflect"
	"testing"

	next "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/v1beta4"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	yaml "gopkg.in/yaml.v2"
)

type testCase struct {
	in       *Config
	expected *next.Config
}

func TestSimple(t *testing.T) {
	testCases := []*testCase{
		{
			in: &Config{
				Deployments: []*DeploymentConfig{
					{
						Name: "Test",
						Component: &ComponentConfig{
							Containers: []*ContainerConfig{
								{
									Name: "container-1",
								},
							},
						},
					},
				},
			},
			expected: &next.Config{
				Deployments: []*next.DeploymentConfig{
					{
						Name: "Test",
						Helm: &next.HelmConfig{
							ComponentChart: ptr.Bool(true),
							Values: map[interface{}]interface{}{
								"containers": []interface{}{
									map[interface{}]interface{}{
										"name": "container-1",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			in: &Config{
				Deployments: []*DeploymentConfig{
					{
						Name: "Test",
						Component: &ComponentConfig{
							Containers: []*ContainerConfig{
								{
									Name: "container-1",
								},
							},
							Options: &ComponentConfigOptions{
								Force: ptr.Bool(true),
								Wait:  ptr.Bool(false),
							},
						},
					},
				},
			},
			expected: &next.Config{
				Deployments: []*next.DeploymentConfig{
					{
						Name: "Test",
						Helm: &next.HelmConfig{
							ComponentChart: ptr.Bool(true),
							Values: map[interface{}]interface{}{
								"containers": []interface{}{
									map[interface{}]interface{}{
										"name": "container-1",
									},
								},
							},
							Force: ptr.Bool(true),
							Wait:  ptr.Bool(false),
						},
					},
				},
			},
		},
		{
			in: &Config{
				Deployments: []*DeploymentConfig{
					{
						Name: "Test",
						Helm: &HelmConfig{
							Chart: &ChartConfig{
								Name:    "component-chart",
								RepoURL: "https://charts.devspace.cloud",
								Version: "v0.0.6",
							},
							Values: map[interface{}]interface{}{
								"containers": []interface{}{
									map[interface{}]interface{}{
										"name": "container-1",
									},
								},
							},
							Force: ptr.Bool(true),
							Wait:  ptr.Bool(false),
						},
					},
				},
			},
			expected: &next.Config{
				Deployments: []*next.DeploymentConfig{
					{
						Name: "Test",
						Helm: &next.HelmConfig{
							ComponentChart: ptr.Bool(true),
							Values: map[interface{}]interface{}{
								"containers": []interface{}{
									map[interface{}]interface{}{
										"name": "container-1",
									},
								},
							},
							Force: ptr.Bool(true),
							Wait:  ptr.Bool(false),
						},
					},
				},
			},
		},
	}

	// Run test cases
	for index, testCase := range testCases {
		newConfig, err := testCase.in.Upgrade(log.Discard)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		isEqual := reflect.DeepEqual(newConfig, testCase.expected)
		if !isEqual {
			newConfigYaml, _ := yaml.Marshal(newConfig)
			expectedYaml, _ := yaml.Marshal(testCase.expected)

			t.Fatalf("TestCase %d: Got %s, but expected %s", index, newConfigYaml, expectedYaml)
		}
	}
}

type testCasePaths struct {
	in       map[string]string
	expected map[string]string
}

func TestUpgradeVarPaths(t *testing.T) {
	config := &Config{}
	testCases := []*testCasePaths{
		{
			in: map[string]string{
				".deployments[1].abc":                   "test1",
				".deployments[1].component.abc":         "test2",
				".deployments[1].component.options.abc": "test3",
				".deployments.notReplace.bcd":           "test4",
				".dev.notreplace":                       "test5",
			},
			expected: map[string]string{
				".deployments[1].abc":             "test1",
				".deployments[1].helm.values.abc": "test2",
				".deployments[1].helm.abc":        "test3",
				".deployments.notReplace.bcd":     "test4",
				".dev.notreplace":                 "test5",
			},
		},
	}

	// Run test cases
	for index, testCase := range testCases {
		err := config.UpgradeVarPaths(testCase.in, log.Discard)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		isEqual := reflect.DeepEqual(testCase.in, testCase.expected)
		if !isEqual {
			newConfigYaml, _ := yaml.Marshal(testCase.in)
			expectedYaml, _ := yaml.Marshal(testCase.expected)

			t.Fatalf("TestCase %d: Got %s, but expected %s", index, newConfigYaml, expectedYaml)
		}
	}
}
