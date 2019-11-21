package loader

import (
	"reflect"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

type testCase struct {
	in       map[interface{}]interface{}
	expected map[interface{}]interface{}
	profile  map[interface{}]interface{}
}

func TestPatches(t *testing.T) {
	testCases := []*testCase{
		{
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
	}

	// Run test cases
	for index, testCase := range testCases {
		newConfig, err := ApplyPatches(testCase.in, testCase.profile)
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
