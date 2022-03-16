package loader

import (
	"reflect"
	"testing"

	yaml "gopkg.in/yaml.v3"
)

func TestTransformPath(t *testing.T) {
	testCases := map[string]string{
		"$.dev": "$.dev",
		".dev":  ".dev",
		"dev":   "dev",
		"deployments.name=backend.helm.values.containers":            "deployments[?(@.name=='backend')].helm.values.containers",
		"deployments.name=backend.helm.values.containers.name=proxy": "deployments[?(@.name=='backend')].helm.values.containers[?(@.name=='proxy')]",
		"/deployments/0":                                     "$.deployments[0]",
		"deployments/0":                                      "deployments[0]",
		"deployments/0/containers/1":                         "deployments[0].containers[1]",
		"deployments.*.containers.*":                         "deployments.*.containers.*",
		"deployments/*/containers/*":                         "deployments[*].containers[*]",
		"deployments/0/containers/1/name":                    "deployments[0].containers[1].name",
		"deployments/*/containers/*/name":                    "deployments[*].containers[*].name",
		"deployments.name=test2":                             "deployments[?(@.name=='test2')]",
		"deployments.name=backend.helm.values.containers[1]": "deployments[?(@.name=='backend')].helm.values.containers[1]",
		`deployments[?(@.name=='staging1')]`:                 `deployments[?(@.name=='staging1')]`,
		`deployments[?(@.helm.timeout > 1000)]`:              `deployments[?(@.helm.timeout > 1000)]`,
		`deployments.name=backend.helm.values.containers.image=john/devbackend.image`: `deployments[?(@.name=='backend')].helm.values.containers[?(@.image=='john/devbackend')].image`,
		`dev.ports.name=rails.reverseForward.port=9200`:                               `dev.ports[?(@.name=='rails')].reverseForward[?(@.port=='9200' || @.port==9200)]`,
	}

	// Run test cases
	for in, expected := range testCases {
		actual := transformPath(in)

		if actual != expected {
			t.Errorf("TestCase %s: Got\n%s, but expected\n%s", in, actual, expected)
		}
	}
}

type testCase struct {
	in       map[string]interface{}
	expected map[string]interface{}
	profile  map[string]interface{}
}

func TestPatches(t *testing.T) {
	testCases := map[string]*testCase{
		"patch with path": {
			profile: map[string]interface{}{
				"name": "test",
				"patches": []interface{}{
					map[string]interface{}{
						"op":   "add",
						"path": "dev.ports",
						"value": map[string]interface{}{
							"imageName": "myImage",
						},
					},
				},
			},
			in: map[string]interface{}{
				"dev": map[string]interface{}{
					"ports": []interface{}{
						map[string]interface{}{
							"imageName": "test",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"dev": map[string]interface{}{
					"ports": []interface{}{
						map[string]interface{}{
							"imageName": "test",
						},
						map[string]interface{}{
							"imageName": "myImage",
						},
					},
				},
			},
		},
		"patch with extended matching": {
			profile: map[string]interface{}{
				"name": "test",
				"patches": []interface{}{
					map[string]interface{}{
						"op":    "add",
						"path":  "dev.ports.imageName=test.containerName",
						"value": "myContainer",
					},
				},
			},
			in: map[string]interface{}{
				"dev": map[string]interface{}{
					"ports": []interface{}{
						map[string]interface{}{
							"imageName": "test",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"dev": map[string]interface{}{
					"ports": []interface{}{
						map[string]interface{}{
							"imageName":     "test",
							"containerName": "myContainer",
						},
					},
				},
			},
		},
		"skip remove patch when no match": {
			profile: map[string]interface{}{
				"name": "test",
				"patches": []interface{}{
					map[string]interface{}{
						"op":   "remove",
						"path": "dev.ports.imageName=test.containerName",
					},
				},
			},
			in: map[string]interface{}{
				"dev": map[string]interface{}{
					"ports": []interface{}{
						map[string]interface{}{
							"imageName": "test",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"dev": map[string]interface{}{
					"ports": []interface{}{
						map[string]interface{}{
							"imageName": "test",
						},
					},
				},
			},
		},
		"add patch when replace patch has no match": {
			profile: map[string]interface{}{
				"name": "test",
				"patches": []interface{}{
					map[string]interface{}{
						"op":   "replace",
						"path": "dev.ports",
						"value": []interface{}{
							map[string]interface{}{
								"imageName": "test",
							},
						},
					},
				},
			},
			in: map[string]interface{}{
				"dev": map[string]interface{}{},
			},
			expected: map[string]interface{}{
				"dev": map[string]interface{}{
					"ports": []interface{}{
						map[string]interface{}{
							"imageName": "test",
						},
					},
				},
			},
		},
		"add patch appends to array without suffix": {
			profile: map[string]interface{}{
				"name": "test",
				"patches": []interface{}{
					map[string]interface{}{
						"op":   "add",
						"path": "dev.ports",
						"value": map[string]interface{}{
							"imageName": "test",
						},
					},
				},
			},
			in: map[string]interface{}{
				"dev": map[string]interface{}{
					"ports": []interface{}{},
				},
			},
			expected: map[string]interface{}{
				"dev": map[string]interface{}{
					"ports": []interface{}{
						map[string]interface{}{
							"imageName": "test",
						},
					},
				},
			},
		},
		"test with wildcard match": {
			profile: map[string]interface{}{
				"name": "test",
				"patches": []interface{}{
					map[string]interface{}{
						"op":    "add",
						"path":  "dev.ports.*.containerName",
						"value": "myContainer",
					},
				},
			},
			in: map[string]interface{}{
				"dev": map[string]interface{}{
					"ports": []interface{}{
						map[string]interface{}{
							"imageName": "test",
						},
						map[string]interface{}{
							"imageName": "myImage",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"dev": map[string]interface{}{
					"ports": []interface{}{
						map[string]interface{}{
							"imageName":     "test",
							"containerName": "myContainer",
						},
						map[string]interface{}{
							"imageName":     "myImage",
							"containerName": "myContainer",
						},
					},
				},
			},
		},
	}

	// Run test cases
	for index, testCase := range testCases {
		newConfig, err := ApplyPatches(testCase.in, testCase.profile)
		if err != nil {
			t.Errorf("Error: %v", err)
		}

		newConfigYaml, _ := yaml.Marshal(newConfig)
		expectedYaml, _ := yaml.Marshal(testCase.expected)
		isEqual := reflect.DeepEqual(newConfigYaml, expectedYaml)
		if !isEqual {
			t.Errorf("TestCase %s: Got\n%s, but expected\n%s", index, newConfigYaml, expectedYaml)
		}
	}
}
