package v1beta10

import (
	"reflect"
	"testing"

	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta11"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"sigs.k8s.io/yaml"
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
				Hooks: []*HookConfig{
					{
						When: &HookWhenConfig{
							After: &HookWhenAtConfig{
								Images:           "all",
								Deployments:      "test",
								Dependencies:     "all",
								PurgeDeployments: "test,test2",
								PullSecrets:      "test",
								InitialSync:      "abc",
							},
							Before: &HookWhenAtConfig{
								Images:           "all",
								Deployments:      "all",
								Dependencies:     "all",
								PurgeDeployments: "test,test2",
								PullSecrets:      "all",
								InitialSync:      "all",
							},
							OnError: &HookWhenAtConfig{
								Images:           "all",
								Deployments:      "all",
								Dependencies:     "test",
								PurgeDeployments: "test,test2",
								PullSecrets:      "all",
								InitialSync:      "all",
							},
						},
					},
				},
			},
			expected: &next.Config{
				Hooks: []*next.HookConfig{
					{
						Events: []string{
							"after:build",
							"after:deploy:test",
							"after:purge:test",
							"after:purge:test2",
							"after:createPullSecrets",
							"after:deployDependencies",
							"after:initialSync:abc",
							"before:build",
							"before:deploy",
							"before:purge:test",
							"before:purge:test2",
							"before:createPullSecrets",
							"before:deployDependencies",
							"before:initialSync:*",
							"error:build:*",
							"error:deploy:*",
							"error:purge:test",
							"error:purge:test2",
							"error:createPullSecrets",
							"error:deployDependencies",
							"error:initialSync:*",
						},
					},
				},
			},
		},
		{
			in: &Config{
				Hooks: []*HookConfig{
					{
						When: &HookWhenConfig{
							After:  &HookWhenAtConfig{},
							Before: &HookWhenAtConfig{},
							OnError: &HookWhenAtConfig{
								PullSecrets: "all",
							},
						},
					},
				},
			},
			expected: &next.Config{
				Hooks: []*next.HookConfig{
					{
						Events: []string{
							"error:createPullSecrets",
						},
					},
				},
			},
		},
		{
			in: &Config{
				Dependencies: []*DependencyConfig{
					{
						Name: "test",
					},
					{
						Name:          "test2",
						OverwriteVars: ptr.Bool(false),
					},
				},
				Commands: []*CommandConfig{
					{
						Name: "test",
					},
					{
						Name:       "test2",
						AppendArgs: ptr.Bool(false),
					},
				},
				Dev: DevConfig{
					Ports: []*PortForwardingConfig{
						{
							ImageName: "test",
						},
						{
							ImageSelector: "test",
						},
					},
					Sync: []*SyncConfig{
						{
							ImageSelector: "test",
						},
						{
							ImageName: "test",
						},
					},
					Logs: &LogsConfig{
						Images: []string{"test", "test3"},
					},
					Terminal: &Terminal{
						ImageName: "terminal",
					},
					ReplacePods: []*ReplacePod{
						{
							ImageSelector: "test",
						},
						{
							ImageName: "test",
						},
					},
				},
			},
			expected: &next.Config{
				Dependencies: []*next.DependencyConfig{
					{
						Name:          "test",
						OverwriteVars: true,
					},
					{
						Name:          "test2",
						OverwriteVars: false,
					},
				},
				Commands: []*next.CommandConfig{
					{
						Name:       "test",
						AppendArgs: true,
					},
					{
						Name:       "test2",
						AppendArgs: false,
					},
				},
				Dev: next.DevConfig{
					Ports: []*next.PortForwardingConfig{
						{
							ImageSelector: "image(test):tag(test)",
						},
						{
							ImageSelector: "test",
						},
					},
					Sync: []*next.SyncConfig{
						{
							ImageSelector: "test",
						},
						{
							ImageSelector: "image(test):tag(test)",
						},
					},
					Logs: &next.LogsConfig{
						Selectors: []next.LogsSelector{
							{
								ImageSelector: "image(test):tag(test)",
							},
							{
								ImageSelector: "image(test3):tag(test3)",
							},
						},
					},
					Terminal: &next.Terminal{
						ImageSelector: "image(terminal):tag(terminal)",
					},
					ReplacePods: []*next.ReplacePod{
						{
							ImageSelector: "test",
						},
						{
							ImageSelector: "image(test):tag(test)",
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
