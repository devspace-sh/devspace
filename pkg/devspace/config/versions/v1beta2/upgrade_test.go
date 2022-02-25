package v1beta2

import (
	"reflect"
	"testing"

	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta3"
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
				Cluster: &Cluster{
					Namespace:   ptr.String("namespace"),
					KubeContext: ptr.String("kubecontext"),
				},
			},
			expected: &next.Config{
				Dev: &next.DevConfig{
					Interactive: &next.InteractiveConfig{
						DefaultEnabled: ptr.Bool(true),
					},
				},
			},
		},
		{
			in: &Config{
				Dev: &DevConfig{
					Selectors: &[]*SelectorConfig{
						{
							Name:          ptr.String("test"),
							Namespace:     ptr.String("namespace"),
							ContainerName: ptr.String("container"),
							LabelSelector: &map[string]*string{
								"my":   ptr.String("app"),
								"test": ptr.String("test"),
							},
						},
					},
					Terminal: &Terminal{
						Selector: ptr.String("test"),
						Command: &[]*string{
							ptr.String("test"),
							ptr.String("test2"),
						},
					},
				},
			},
			expected: &next.Config{
				Dev: &next.DevConfig{
					Interactive: &next.InteractiveConfig{
						DefaultEnabled: ptr.Bool(true),
						Terminal: &next.TerminalConfig{
							Namespace:     "namespace",
							ContainerName: "container",
							LabelSelector: map[string]string{
								"my":   "app",
								"test": "test",
							},
							Command: []string{"test", "test2"},
						},
					},
				},
			},
		},
		{
			in: &Config{
				Dev: &DevConfig{
					OverrideImages: &[]*ImageOverrideConfig{
						{
							Name:       ptr.String("test"),
							Entrypoint: &[]*string{ptr.String("my"), ptr.String("command")},
						},
					},
					Selectors: &[]*SelectorConfig{
						{
							Name:          ptr.String("test"),
							Namespace:     ptr.String("namespace"),
							ContainerName: ptr.String("container"),
							LabelSelector: &map[string]*string{
								"my":   ptr.String("app"),
								"test": ptr.String("test"),
							},
						},
					},
					Ports: &[]*PortForwardingConfig{
						{
							Selector: ptr.String("test"),
						},
					},
					Sync: &[]*SyncConfig{
						{
							Selector: ptr.String("test"),
						},
					},
					Terminal: &Terminal{
						Disabled: ptr.Bool(true),
					},
				},
				Images: &map[string]*ImageConfig{
					"default": {},
				},
			},
			expected: &next.Config{
				Dev: &next.DevConfig{
					Interactive: &next.InteractiveConfig{
						DefaultEnabled: ptr.Bool(false),
						Images: []*next.InteractiveImageConfig{
							{
								Name:       "test",
								Entrypoint: []string{"my"},
								Cmd:        []string{"command"},
							},
						},
					},
					Ports: []*next.PortForwardingConfig{
						{
							Namespace: "namespace",
							LabelSelector: map[string]string{
								"my":   "app",
								"test": "test",
							},
						},
					},
					Sync: []*next.SyncConfig{
						{
							Namespace:     "namespace",
							ContainerName: "container",
							LabelSelector: map[string]string{
								"my":   "app",
								"test": "test",
							},
						},
					},
				},
				Images: map[string]*next.ImageConfig{
					"default": {
						CreatePullSecret: ptr.Bool(false),
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
