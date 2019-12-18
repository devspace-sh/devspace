package v1beta5

import (
	"reflect"
	"testing"

	next "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
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
				Images: map[string]*ImageConfig{
					"test": &ImageConfig{
						Image: "Test",
						Build: &BuildConfig{
							Disabled: ptr.Bool(true),
							Custom: &CustomConfig{
								Command: "MyCommand",
								Args:    []*string{ptr.String("Test"), ptr.String("Test2")},
							},
						},
					},
					"test2": &ImageConfig{
						Image: "Test",
					},
					"test3": &ImageConfig{
						Image: "Test",
						Build: &BuildConfig{
							Custom: &CustomConfig{
								Args: []*string{ptr.String("Test"), ptr.String("Test2")},
							},
						},
					},
				},
			},
			expected: &next.Config{
				Images: map[string]*next.ImageConfig{
					"test": &next.ImageConfig{
						Image: "Test",
						Build: &next.BuildConfig{
							Disabled: ptr.Bool(true),
							Custom: &next.CustomConfig{
								Command: "MyCommand",
								Args:    []*string{ptr.String("Test"), ptr.String("Test2")},
							},
						},
					},
					"test2": &next.ImageConfig{
						Image: "Test",
					},
					"test3": &next.ImageConfig{
						Image: "Test",
						Build: &next.BuildConfig{
							Custom: &next.CustomConfig{
								Args: []*string{ptr.String("Test"), ptr.String("Test2")},
							},
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
