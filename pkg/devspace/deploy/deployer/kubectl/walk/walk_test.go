package walk

import (
	"testing"

	"gopkg.in/yaml.v3"
	"gotest.tools/assert"
)

func TestWalk(t *testing.T) {

	// Input yaml
	input := `
test:
    image: appendtag
    test: []
test2:
    image: dontreplaceme
    test3:
    - test4:
        test5:
        image: replaceme
`
	inputObj := make(map[string]interface{})
	err := yaml.Unmarshal([]byte(input), inputObj)
	if err != nil {
		t.Fatalf("Error parsing input: %v", err)
	}

	match := MatchFn(func(key, value string) bool {
		return key == "image" && value != "dontreplaceme"
	})
	replace := ReplaceFn(func(_, value string) (interface{}, error) {
		if value == "appendtag" {
			return "appendtag:test", nil
		}

		return "replaced", nil
	})

	_ = Walk(inputObj, match, replace)

	output, err := yaml.Marshal(inputObj)
	if err != nil {
		t.Fatalf("Error parsing output: %v", err)
	}

	// Output yaml
	expected := `test:
    image: appendtag:test
    test: []
test2:
    image: dontreplaceme
    test3:
        - test4:
            image: replaced
            test5: null
`
	assert.Equal(t, string(output), expected)
}
