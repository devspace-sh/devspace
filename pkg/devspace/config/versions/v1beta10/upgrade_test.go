package v1beta10

import (
	"github.com/ghodss/yaml"
	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
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
				Hooks: []*HookConfig{
					{
						When: &HookWhenConfig{
							After: &HookWhenAtConfig{
								Images:           "all",
								Deployments:      "test",
								PurgeDeployments: "test,test2",
								PullSecrets:      "test",
								InitialSync:      "abc",
							},
							Before: &HookWhenAtConfig{
								Images:           "all",
								Deployments:      "all",
								PurgeDeployments: "test,test2",
								PullSecrets:      "all",
								InitialSync:      "all",
							},
							OnError: &HookWhenAtConfig{
								Images:           "all",
								Deployments:      "test",
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
							"after:buildAll",
							"after:deploy:test",
							"after:purge:test",
							"after:purge:test2",
							"after:createAllPullSecrets",
							"after:initialSync:abc",
							"before:buildAll",
							"before:deployAll",
							"before:purge:test",
							"before:purge:test2",
							"before:createAllPullSecrets",
							"before:initialSync:*",
							"error:buildAll",
							"error:deploy:test",
							"error:purge:test",
							"error:purge:test2",
							"error:createAllPullSecrets",
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
							After: &HookWhenAtConfig{},
							Before: &HookWhenAtConfig{},
							OnError: &HookWhenAtConfig{
								PullSecrets:      "all",
							},
						},
					},
				},
			},
			expected: &next.Config{
				Hooks: []*next.HookConfig{
					{
						Events: []string{
							"error:createAllPullSecrets",
						},
					},
				},
			},
		},
		{
			in: &Config{
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
