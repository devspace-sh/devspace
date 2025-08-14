package loader

import (
	"reflect"
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	yaml "gopkg.in/yaml.v3"
)

type testCase struct {
	in       map[string]interface{}
	expected map[string]interface{}
	profile  latest.ProfileConfig
}

func TestPatches(t *testing.T) {
	testCases := map[string]*testCase{
		"patch with path": {
			profile: latest.ProfileConfig{
				Name: "test",
				Patches: []*latest.PatchConfig{
					{
						Operation: "add",
						Path:      "dev.ports",
						Value: map[string]interface{}{
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
			profile: latest.ProfileConfig{
				Name: "test",
				Patches: []*latest.PatchConfig{
					{
						Operation: "add",
						Path:      "dev.ports.imageName=test.containerName",
						Value:     "myContainer",
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
			profile: latest.ProfileConfig{
				Name: "test",
				Patches: []*latest.PatchConfig{
					{
						Operation: "remove",
						Path:      "dev.ports.imageName=test.containerName",
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
			profile: latest.ProfileConfig{
				Name: "test",
				Patches: []*latest.PatchConfig{
					{
						Operation: "replace",
						Path:      "dev.ports",
						Value: []interface{}{
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
			profile: latest.ProfileConfig{
				Name: "test",
				Patches: []*latest.PatchConfig{
					{
						Operation: "add",
						Path:      "dev.ports",
						Value: map[string]interface{}{
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
			profile: latest.ProfileConfig{
				Name: "test",
				Patches: []*latest.PatchConfig{
					{
						Operation: "add",
						Path:      "dev.ports.*.containerName",
						Value:     "myContainer",
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
		newConfig, err := ApplyPatches(testCase.in, &testCase.profile)
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
