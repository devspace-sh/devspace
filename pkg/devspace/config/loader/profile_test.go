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
	in       map[interface{}]interface{}
	expected map[interface{}]interface{}
	profile  map[interface{}]interface{}
}

func TestPatches(t *testing.T) {
	testCases := map[string]*testCase{
		"patch with path": {
			profile: map[interface{}]interface{}{
				"name": "test",
				"patches": []interface{}{
					map[interface{}]interface{}{
						"op":   "add",
						"path": "dev.ports",
						"value": map[interface{}]interface{}{
							"imageName": "myImage",
						},
					},
				},
			},
			in: map[interface{}]interface{}{
				"dev": map[interface{}]interface{}{
					"ports": []interface{}{
						map[interface{}]interface{}{
							"imageName": "test",
						},
					},
				},
			},
			expected: map[interface{}]interface{}{
				"dev": map[interface{}]interface{}{
					"ports": []interface{}{
						map[interface{}]interface{}{
							"imageName": "test",
						},
						map[interface{}]interface{}{
							"imageName": "myImage",
						},
					},
				},
			},
		},
		"patch with extended matching": {
			profile: map[interface{}]interface{}{
				"name": "test",
				"patches": []interface{}{
					map[interface{}]interface{}{
						"op":    "add",
						"path":  "dev.ports.imageName=test.containerName",
						"value": "myContainer",
					},
				},
			},
			in: map[interface{}]interface{}{
				"dev": map[interface{}]interface{}{
					"ports": []interface{}{
						map[interface{}]interface{}{
							"imageName": "test",
						},
					},
				},
			},
			expected: map[interface{}]interface{}{
				"dev": map[interface{}]interface{}{
					"ports": []interface{}{
						map[interface{}]interface{}{
							"imageName":     "test",
							"containerName": "myContainer",
						},
					},
				},
			},
		},
		"skip remove patch when no match": {
			profile: map[interface{}]interface{}{
				"name": "test",
				"patches": []interface{}{
					map[interface{}]interface{}{
						"op":   "remove",
						"path": "dev.ports.imageName=test.containerName",
					},
				},
			},
			in: map[interface{}]interface{}{
				"dev": map[interface{}]interface{}{
					"ports": []interface{}{
						map[interface{}]interface{}{
							"imageName": "test",
						},
					},
				},
			},
			expected: map[interface{}]interface{}{
				"dev": map[interface{}]interface{}{
					"ports": []interface{}{
						map[interface{}]interface{}{
							"imageName": "test",
						},
					},
				},
			},
		},
		"add patch when replace patch has no match": {
			profile: map[interface{}]interface{}{
				"name": "test",
				"patches": []interface{}{
					map[interface{}]interface{}{
						"op":   "replace",
						"path": "dev.ports",
						"value": []interface{}{
							map[interface{}]interface{}{
								"imageName": "test",
							},
						},
					},
				},
			},
			in: map[interface{}]interface{}{
				"dev": map[interface{}]interface{}{},
			},
			expected: map[interface{}]interface{}{
				"dev": map[interface{}]interface{}{
					"ports": []interface{}{
						map[interface{}]interface{}{
							"imageName": "test",
						},
					},
				},
			},
		},
		"add patch appends to array without suffix": {
			profile: map[interface{}]interface{}{
				"name": "test",
				"patches": []interface{}{
					map[interface{}]interface{}{
						"op":   "add",
						"path": "dev.ports",
						"value": map[interface{}]interface{}{
							"imageName": "test",
						},
					},
				},
			},
			in: map[interface{}]interface{}{
				"dev": map[interface{}]interface{}{
					"ports": []interface{}{},
				},
			},
			expected: map[interface{}]interface{}{
				"dev": map[interface{}]interface{}{
					"ports": []interface{}{
						map[interface{}]interface{}{
							"imageName": "test",
						},
					},
				},
			},
		},
		"test with wildcard match": {
			profile: map[interface{}]interface{}{
				"name": "test",
				"patches": []interface{}{
					map[interface{}]interface{}{
						"op":    "add",
						"path":  "dev.ports.*.containerName",
						"value": "myContainer",
					},
				},
			},
			in: map[interface{}]interface{}{
				"dev": map[interface{}]interface{}{
					"ports": []interface{}{
						map[interface{}]interface{}{
							"imageName": "test",
						},
						map[interface{}]interface{}{
							"imageName": "myImage",
						},
					},
				},
			},
			expected: map[interface{}]interface{}{
				"dev": map[interface{}]interface{}{
					"ports": []interface{}{
						map[interface{}]interface{}{
							"imageName":     "test",
							"containerName": "myContainer",
						},
						map[interface{}]interface{}{
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
