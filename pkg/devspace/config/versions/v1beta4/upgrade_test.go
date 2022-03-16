package v1beta4

import (
	"reflect"
	"testing"

	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta5"
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
						Helm: &HelmConfig{
							Rollback: ptr.Bool(true),
						},
					},
				},
			},
			expected: &next.Config{
				Deployments: []*next.DeploymentConfig{
					{
						Name: "Test",
						Helm: &next.HelmConfig{
							V2:     true,
							Atomic: true,
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
