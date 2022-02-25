package v1beta1

import (
	"reflect"
	"testing"

	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta2"
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
				Images: &map[string]*ImageConfig{
					"test-1": {
						Image: ptr.String("test-1"),
					},
				},
			},
			expected: &next.Config{
				Images: &map[string]*next.ImageConfig{
					"test-1": {
						Image: ptr.String("test-1"),
					},
				},
			},
		},
		{
			in: &Config{
				Images: &map[string]*ImageConfig{
					"test-1": {
						Image: ptr.String("test-1"),
						Build: &BuildConfig{
							Dockerfile: ptr.String("dockerfile"),
						},
					},
				},
			},
			expected: &next.Config{
				Images: &map[string]*next.ImageConfig{
					"test-1": {
						Image:      ptr.String("test-1"),
						Dockerfile: ptr.String("dockerfile"),
						Build:      &next.BuildConfig{},
					},
				},
			},
		},
		{
			in: &Config{
				Images: &map[string]*ImageConfig{
					"test-1": {
						Image: ptr.String("test-1"),
						Build: &BuildConfig{
							Dockerfile: ptr.String("dockerfile"),
						},
					},
				},
			},
			expected: &next.Config{
				Images: &map[string]*next.ImageConfig{
					"test-1": {
						Image:      ptr.String("test-1"),
						Dockerfile: ptr.String("dockerfile"),
						Build:      &next.BuildConfig{},
					},
				},
			},
		},
		{
			in: &Config{
				Images: &map[string]*ImageConfig{
					"test-1": {
						Image:    ptr.String("test-1"),
						Insecure: ptr.Bool(true),
						SkipPush: ptr.Bool(true),
					},
				},
			},
			expected: &next.Config{
				Images: &map[string]*next.ImageConfig{
					"test-1": {
						Image: ptr.String("test-1"),
						Build: &next.BuildConfig{
							Kaniko: &next.KanikoConfig{
								Insecure: ptr.Bool(true),
							},
							Docker: &next.DockerConfig{
								SkipPush: ptr.Bool(true),
							},
						},
					},
				},
			},
		},
		{
			in: &Config{
				Images: &map[string]*ImageConfig{
					"test-1": {
						Image:    ptr.String("test-1"),
						Insecure: ptr.Bool(true),
						SkipPush: ptr.Bool(true),
						Build: &BuildConfig{
							Options: &BuildOptions{
								Network: ptr.String("test-network"),
								Target:  ptr.String("target"),
								BuildArgs: &map[string]*string{
									"test-arg1": ptr.String("test-value1"),
									"test-arg2": ptr.String("test-value2"),
								},
							},
						},
					},
				},
			},
			expected: &next.Config{
				Images: &map[string]*next.ImageConfig{
					"test-1": {
						Image: ptr.String("test-1"),
						Build: &next.BuildConfig{
							Kaniko: &next.KanikoConfig{
								Insecure: ptr.Bool(true),
								Options: &next.BuildOptions{
									Network: ptr.String("test-network"),
									Target:  ptr.String("target"),
									BuildArgs: &map[string]*string{
										"test-arg1": ptr.String("test-value1"),
										"test-arg2": ptr.String("test-value2"),
									},
								},
							},
							Docker: &next.DockerConfig{
								SkipPush: ptr.Bool(true),
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
