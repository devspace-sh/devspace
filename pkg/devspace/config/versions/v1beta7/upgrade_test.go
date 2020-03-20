package v1beta7

import (
	next "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/ghodss/yaml"
	"reflect"
	"testing"
)

type testCase struct {
	in       *Config
	expected *next.Config
}

func TestSimple(t *testing.T) {
	testCases := []*testCase{
		{
			in:       &Config{},
			expected: &next.Config{},
		},
		{
			in: &Config{
				Images: map[string]*ImageConfig{
					"test": {
						Build: &BuildConfig{
							Custom: &CustomConfig{
								ImageFlag: "test",
							},
						},
					},
				},
			},
			expected: &next.Config{
				Images: map[string]*next.ImageConfig{
					"test": {
						Build: &next.BuildConfig{
							Custom: &next.CustomConfig{
								ImageArg: "test",
							},
						},
					},
				},
			},
		},
		{
			in: &Config{
				Images: map[string]*ImageConfig{
					"test": {
						Build: &BuildConfig{
							Kaniko: &KanikoConfig{
								Flags: []string{"test", "test2"},
							},
						},
					},
				},
			},
			expected: &next.Config{
				Images: map[string]*next.ImageConfig{
					"test": {
						Build: &next.BuildConfig{
							Kaniko: &next.KanikoConfig{
								Args: []string{"test", "test2"},
							},
						},
					},
				},
			},
		},
		{
			in: &Config{
				Deployments: []*DeploymentConfig{
					&DeploymentConfig{
						Kubectl: &KubectlConfig{
							Flags: []string{"test", "test2"},
						},
					},
				},
			},
			expected: &next.Config{
				Deployments: []*next.DeploymentConfig{
					&next.DeploymentConfig{
						Kubectl: &next.KubectlConfig{
							ApplyArgs: []string{"test", "test2"},
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
