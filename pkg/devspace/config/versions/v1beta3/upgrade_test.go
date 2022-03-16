package v1beta3

import (
	"reflect"
	"testing"

	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta4"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	yaml "gopkg.in/yaml.v3"
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
							Values: map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
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
							Values: map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
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
							Values: map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
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
							Values: map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
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
