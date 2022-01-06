package patch

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v3"
)

type operationTestCase struct {
	operation   *Operation
	expected    string
	expectedErr error
	input       string
}

func TestAddOperation(t *testing.T) {
	testCases := map[string]*operationTestCase{
		"adding an element to an object": {
			input: "foo: bar",
			operation: &Operation{
				Op:    opAdd,
				Path:  ".baz",
				Value: loadYamlFromString(`qux`),
			},
			expected: `
				foo: bar
				baz: qux
			`,
		},
		"adding an element to an array": {
			input: "foo: [bar,baz]",
			operation: &Operation{
				Op:    opAdd,
				Path:  ".foo[1]",
				Value: loadYamlFromString(`qux`),
			},
			expected: "foo: [bar,qux,baz]",
		},
		"adding an object to an object": {
			input: `
				foo: bar
			`,
			operation: &Operation{
				Op:   opAdd,
				Path: ".child",
				Value: loadYamlFromString(`
					grandchild: {}
				`),
			},
			expected: `
				foo: bar
				child:
					grandchild: {}
			`,
		},
		"appending an element to an array": {
			input: `foo: [bar]`,
			operation: &Operation{
				Op:    opAdd,
				Path:  ".foo",
				Value: loadYamlFromString(`[abc,def]`),
			},
			expected: `foo: [bar, [abc, def]]`,
		},
		"adding a nil element to an object": {
			input: `
				foo: bar
			`,
			operation: &Operation{
				Op:    opAdd,
				Path:  ".baz",
				Value: loadYamlFromString(`~`),
			},
			expected: `
				foo: bar
				baz: ~
			`,
		},
		"adding an element to the root of a document": {
			input: `
				foo: bar
			`,
			operation: &Operation{
				Op:   opAdd,
				Path: "$",
				Value: loadYamlFromString(`
					baz: qux
				`),
			},
			expected: `
				foo: bar
				baz: qux
			`,
		},
		"adding array to missing path with parent": {
			input: `
				dev: {}
			`,
			operation: &Operation{
				Op:   opAdd,
				Path: ".dev.ports",
				Value: loadYamlFromString(`
					- imageName: test
				`),
			},
			expected: `
				dev: {ports: [{imageName: test}]}
			`,
		},
		"add property to object": {
			input: `
				images:
					backend:
						image: john/devbackend
			`,
			operation: &Operation{
				Op:   opAdd,
				Path: ".images.backend",
				Value: loadYamlFromString(`
					dockerfile: ./prod.Dockerfile
				`),
			},
			expected: `
				images:
					backend:
						image: john/devbackend
						dockerfile: ./prod.Dockerfile
			`,
		},
		"add to map": {
			input: `
				images:
					backend:
						image: john/devbackend
			`,
			operation: &Operation{
				Op:   opAdd,
				Path: ".images",
				Value: loadYamlFromString(`
					backend-prod:
						image: john/prodbackend
						dockerfile: ./prod.Dockerfile
				`),
			},
			expected: `
				images:
					backend:
						image: john/devbackend
					backend-prod:
						image: john/prodbackend
						dockerfile: ./prod.Dockerfile
			`,
		},
		"add to array": {
			input: `
				deployments:
					- name: devbackend
			`,
			operation: &Operation{
				Op:   opAdd,
				Path: ".deployments",
				Value: loadYamlFromString(`
					name: prodbackend
				`),
			},
			expected: `
				deployments:
					- name: devbackend
					- name: prodbackend
			`,
		},
		"add to array that doesn't exist": {
			input: `
				version: v1beta10
			`,
			operation: &Operation{
				Op:   opAdd,
				Path: ".deployments",
				Value: loadYamlFromString(`
					- name: prodbackend
				`),
			},
			expected: `
				version: v1beta10
				deployments:
					- name: prodbackend
			`,
		},
		"add to object that doesn't exist": {
			input: `
				deployments:
					- name: devbackend
			`,
			operation: &Operation{
				Op:   opAdd,
				Path: ".new",
				Value: loadYamlFromString(`
					name: prodbackend
				`),
			},
			expected: `
				deployments:
					- name: devbackend
				new:
					name: prodbackend
			`,
		},
		"add to wildcard path that doesn't exist": {
			input: `
				deployments:
					- name: devbackend
					- name: prodbackend
			`,
			operation: &Operation{
				Op:   opAdd,
				Path: "deployments[*].helm",
				Value: loadYamlFromString(`
					values:
						containers:
							- name: proxy
				`),
			},
			expected: `
				deployments:
					- name: devbackend
					  helm:
						values:
							containers:
								- name: proxy
					- name: prodbackend
					  helm:
						values:
							containers:
								- name: proxy
			`,
		},
		"add to wildcard path where some exist": {
			input: `
				deployments:
					- name: devbackend
					- name: prodbackend
					  kubectl:
					  	manifests: []
			`,
			operation: &Operation{
				Op:   opAdd,
				Path: "deployments[*].kubectl.manifests",
				Value: loadYamlFromString(`
					"test.yaml"
				`),
			},
			expected: `
				deployments:
					- name: devbackend
					- name: prodbackend
					  kubectl:
						manifests: ["test.yaml"]
			`,
		},
	}

	runTestCases(testCases, t)
}

func TestRemoveOperation(t *testing.T) {
	testCases := map[string]*operationTestCase{
		"removing an element from an object": {
			input: `
				foo: bar
				baz: qux
			`,
			operation: &Operation{
				Op:   opRemove,
				Path: ".baz",
			},
			expected: `
				foo: bar
			`,
		},
		"removing an element from an array": {
			input: `foo: [bar,qux,baz]`,
			operation: &Operation{
				Op:   opRemove,
				Path: ".foo[1]",
			},
			expected: `foo: [bar,baz]`,
		},
		"removing a nil element from an object": {
			input: `
				foo: bar
				qux:
					baz: 1
					bar: ~
			`,
			operation: &Operation{
				Op:   opRemove,
				Path: ".qux.bar",
			},
			expected: `
				foo: bar
				qux:
					baz: 1
			`,
		},
		"remove from array by filter": {
			input: `
				deployments:
					- name: devbackend
					- name: prodbackend
			`,
			operation: &Operation{
				Op:   opRemove,
				Path: ".deployments[?(@.name=='prodbackend')]",
			},
			expected: `
				deployments:
					- name: devbackend
			`,
		},
		"remove nested item matched by numeric property from array by string or numeric filter": {
			input: `
                dev:
                    ports:
                    - name: rails
                      reverseForward:
                      - port: 9200
                        remotePort: 9200
			`,
			operation: &Operation{
				Op:   opRemove,
				Path: "dev.ports[?(@.name=='rails')].reverseForward[?(@.port==9200 || @.port=='9200')]",
			},
			expected: `
                dev:
                    ports:
                    - name: rails
                      reverseForward: []
			`,
		},
		"remove nested item matched by string property from array by string or numeric filter": {
			input: `
                dev:
                    ports:
                    - name: rails
                      reverseForward:
                      - port: '9200'
                        remotePort: '9200'
			`,
			operation: &Operation{
				Op:   opRemove,
				Path: "dev.ports[?(@.name=='rails')].reverseForward[?(@.port==9200 || @.port=='9200')]",
			},
			expected: `
				dev:
                    ports:
                    - name: rails
                      reverseForward: []
			`,
		},
		"remove no match": {
			input: `
				deployments:
					- name: devbackend
					- name: prodbackend
			`,
			operation: &Operation{
				Op:   opRemove,
				Path: ".deployments[?(@.name=='nonexisting')]",
			},
			expectedErr: fmt.Errorf("remove operation does not apply: doc is missing path: .deployments[?(@.name=='nonexisting')]"),
		},
	}

	runTestCases(testCases, t)
}

func TestReplaceOperation(t *testing.T) {
	testCases := map[string]*operationTestCase{
		"replacing an element in an object": {
			input: `
				foo: bar
				baz: qux
			`,
			operation: &Operation{
				Op:    opReplace,
				Path:  "baz",
				Value: loadYamlFromString(`boo`),
			},
			expected: `
				foo: bar
				baz: boo
			`,
		},
		"replacing the sole element in an array": {
			input: `foo: [bar]`,
			operation: &Operation{
				Op:    opReplace,
				Path:  ".foo[0]",
				Value: loadYamlFromString(`baz`),
			},
			expected: `foo: [baz]`,
		},
		"replacing an element in an array within a root array": {
			input: `- foo: [bar, qux, baz]`,
			operation: &Operation{
				Op:    opReplace,
				Path:  "[0].foo[0]",
				Value: loadYamlFromString(`bum`),
			},
			expected: `- foo: [bum, qux, baz]`,
		},
		"replacing an element in the root object": {
			input: `
				foo: bar
			`,
			operation: &Operation{
				Op:    opReplace,
				Path:  ".foo",
				Value: loadYamlFromString(`qux`),
			},
			expected: `
				foo: qux
			`,
		},
		"replace scalar value": {
			input: `
				images:
					backend:
						image: john/devbackend
					backend-prod:
						image: john/prodbackend
						dockerfile: ./prod.Dockerfile
			`,
			operation: &Operation{
				Op:   opReplace,
				Path: "images.backend.image",
				Value: loadYamlFromString(`
					john/stagingbackend
				`),
			},
			expected: `
				images:
					backend:
						image: john/stagingbackend
					backend-prod:
						image: john/prodbackend
						dockerfile: ./prod.Dockerfile
			`,
		},
		"replace array by name": {
			input: `
				deployments:
					- name: devbackend
					- name: prodbackend
			`,
			operation: &Operation{
				Op:   opReplace,
				Path: ".deployments[?(@.name=='prodbackend')]",
				Value: loadYamlFromString(`
					name: testbackend
				`),
			},
			expected: `
				deployments:
					- name: devbackend
					- name: testbackend
			`,
		},
		"replace object": {
			input: `
				images:
					backend:
						image: john/devbackend
					backend-prod:
						image: john/prodbackend
						dockerfile: ./prod.Dockerfile
			`,
			operation: &Operation{
				Op:   opReplace,
				Path: ".images.backend",
				Value: loadYamlFromString(`
					image: john/stagingbackend
				`),
			},
			expected: `
				images:
					backend:
						image: john/stagingbackend
					backend-prod:
						image: john/prodbackend
						dockerfile: ./prod.Dockerfile
			`,
		},
		"replace all property values": {
			input: `
				images:
					backend:
						image: john/devbackend
					backend-prod:
						image: john/prodbackend
						dockerfile: ./prod.Dockerfile
			`,
			operation: &Operation{
				Op:   opReplace,
				Path: ".images.*.image",
				Value: loadYamlFromString(`
					john/stagingbackend
				`),
			},
			expected: `
				images:
					backend:
						image: john/stagingbackend
					backend-prod:
						image: john/stagingbackend
						dockerfile: ./prod.Dockerfile
			`,
		},

		"replace all objects": {
			input: `
				images:
					backend:
						image: john/devbackend
					backend-prod:
						image: john/prodbackend
						dockerfile: ./prod.Dockerfile
			`,
			operation: &Operation{
				Op:   opReplace,
				Path: ".images.*",
				Value: loadYamlFromString(`
					image: john/stagingbackend
				`),
			},
			expected: `
				images:
					backend:
						image: john/stagingbackend
					backend-prod:
						image: john/stagingbackend
			`,
		},
		"replace no match": {
			input: `
				deployments:
					- name: devbackend
					- name: prodbackend
			`,
			operation: &Operation{
				Op:   opReplace,
				Path: ".images.backend-nonexisting",
				Value: loadYamlFromString(`
					image: john/stagingbackend
				`),
			},
			expectedErr: fmt.Errorf("replace operation does not apply: doc is missing path: .images.backend-nonexisting"),
		},
	}

	runTestCases(testCases, t)
}

func convertTabs(str string) string {
	return strings.ReplaceAll(str, "\t", "    ")
}

func loadYamlFromString(str string) *yaml.Node {
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(convertTabs(str)), &node); err != nil {
		fmt.Println(str)
		fmt.Println(err)
	}
	return &node
}

func runTestCases(testCases map[string]*operationTestCase, t *testing.T) {
	for name, testCase := range testCases {
		inputNode := loadYamlFromString(testCase.input)

		err := testCase.operation.Perform(inputNode)
		if err != nil {
			if testCase.expectedErr != nil && reflect.DeepEqual(err.Error(), testCase.expectedErr.Error()) {
				continue
			}
			t.Errorf("Error %v in case %s", err, name)
		}

		actual, err := yaml.Marshal(inputNode)
		if err != nil {
			t.Errorf("Error %v in case %s", err, name)
		}
		actualStr := string(actual)

		expectedNode := loadYamlFromString(testCase.expected)
		expected, err := yaml.Marshal(expectedNode)
		if err != nil {
			t.Errorf("Error %v in case %s", err, name)
		}
		expectedStr := string(expected)

		isEqual := reflect.DeepEqual(actualStr, expectedStr)
		if !isEqual {
			t.Errorf("TestCase %s\n\nactual:\n\n%s\nbut expected:\n\n%s", name, actualStr, expectedStr)
		}
	}
}
