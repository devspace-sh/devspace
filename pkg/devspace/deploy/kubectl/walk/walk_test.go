package walk

import (
	"testing"
	"gotest.tools/assert" 
"gopkg.in/yaml.v2"
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
	inputObj := make(map[interface{}]interface{})
	err := yaml.Unmarshal([]byte(input), inputObj)
	if err != nil {
		t.Fatalf("Error parsing input: %v", err)
	}

	match := MatchFn(func(path, key, value string) bool{
		return key == "image" && value != "dontreplaceme"
	})
	replace := ReplaceFn(func(path, value string) (interface{}){
		if value == "appendtag" {
			return "appendtag:test"
		}else {
			return "replaced"
		}
	})

	Walk(inputObj, match, replace)

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

		t.Log(string(output))
	assert.Equal(t, string(output), expected)
}
