package patch

import (
	"testing"
)

func TestIsRootChild(t *testing.T) {
	testCases := map[OpPath]bool{
		"":      false,
		".baz":  true,
		"$baz":  true,
		"$":     false,
		"*.baz": false,
	}

	for input, expected := range testCases {
		actual := input.isRootChild()
		if expected != actual {
			t.Errorf("TestCase %s\nactual:%t\nexpected:%t", input, actual, expected)
		}
	}
}

func TestGetChildName(t *testing.T) {
	testCases := map[OpPath]string{
		``:                   ``,
		`.baz`:               `baz`,
		`$baz`:               `baz`,
		`$`:                  ``,
		`*.baz`:              `baz`,
		`*..baz`:             `baz`,
		"deployments['baz']": `baz`,
		`deployments["baz"]`: `baz`,
	}

	for input, expected := range testCases {
		actual := input.getChildName()
		if expected != actual {
			t.Errorf("TestCase %s\nactual:%s\nexpected:%s", input, actual, expected)
		}
	}
}

func TestGetParentPath(t *testing.T) {
	testCases := map[OpPath]string{
		"":                                "",
		"parent1.child1":                  "parent1",
		"parent1['child1']":               "parent1",
		"parent1.parent2.child1":          "parent1.parent2",
		"parent1['parent2']['child1']":    "parent1['parent2']",
		"$.parent1.child1":                "$.parent1",
		"$.*.child1":                      "$.*",
		"$.deployments[*].parent1.child1": "$.deployments[*].parent1",
		"$.deployments[?(@.name=='backend')].parent1.child1":  "$.deployments[?(@.name=='backend')].parent1",
		"$.deployments[?(@.name=~/^backend/)].parent1.child1": "$.deployments[?(@.name=~/^backend/)].parent1",
		"$.deployments[?(@.name=~/^backend/)].child1":         "$.deployments[?(@.name=~/^backend/)]",
		"$.deployments[?(@.name=~/^Backend/)].child1":         "$.deployments[?(@.name=~/^Backend/)]",
		"$.deployments[?(@.name=~/^backend/)]":                "$.deployments",
		"$.deployments[?(@.name=~/^backend\\//)]":             "$.deployments",
		"$.deployments[*]": "$.deployments",
	}

	for input, expected := range testCases {
		actual := input.getParentPath()
		if expected != actual {
			t.Errorf("TestCase %s\nactual:%s\nexpected:%s", input, actual, expected)
		}
	}
}

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
		actual := TransformPath(in)

		if actual != expected {
			t.Errorf("TestCase %s: Got\n%s, but expected\n%s", in, actual, expected)
		}
	}
}
