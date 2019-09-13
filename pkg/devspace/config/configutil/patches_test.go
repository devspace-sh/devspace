package configutil

import (
	"reflect"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	yaml "gopkg.in/yaml.v2"
)

type testCase struct {
	in       *latest.Config
	expected *latest.Config
}

func TestPatches(t *testing.T) {
	testCases := []*testCase{
		{
			in: &latest.Config{
				Dev: &latest.DevConfig{
					Ports: []*latest.PortForwardingConfig{
						{
							ImageName: "test",
						},
					},
				},
				Profiles: []*latest.ProfileConfig{
					{
						Name: "test",
						Patches: []*latest.PatchConfig{
							{
								Operation: "add",
								Path:      "dev.ports",
								Value: map[interface{}]interface{}{
									"imageName": "myImage",
								},
							},
						},
					},
				},
			},
			expected: &latest.Config{
				Dev: &latest.DevConfig{
					Ports: []*latest.PortForwardingConfig{
						{
							ImageName: "test",
						},
						{
							ImageName: "myImage",
						},
					},
				},
				Profiles: []*latest.ProfileConfig{
					{
						Name: "test",
						Patches: []*latest.PatchConfig{
							{
								Operation: "add",
								Path:      "dev.ports",
								Value: map[interface{}]interface{}{
									"imageName": "myImage",
								},
							},
						},
					},
				},
			},
		},
	}

	// Run test cases
	for index, testCase := range testCases {
		newConfig, err := ApplyPatches(testCase.in)
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
