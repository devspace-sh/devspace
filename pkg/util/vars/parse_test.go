package vars

import (
	"errors"
	"testing"

	"github.com/loft-sh/devspace/pkg/util/ptr"
)

type testCase struct {
	input   string
	replace ReplaceVarFn
	output  interface{}
	err     *string
}

func TestParse(t *testing.T) {
	testCases := map[string]*testCase{
		"Single Replace": {
			input:   " test abc ${Test} ",
			replace: func(value string) (interface{}, error) { return "test", nil },
			output:  " test abc test ",
		},
		"Var Name 2": {
			input: " test abc ${Test} ",
			replace: func(value string) (interface{}, error) {
				if value != "Test" {
					return "", errors.New("unexpected var name")
				}
				return "test", nil
			},
			output: " test abc test ",
		},
		"Var Name": {
			input: " test abc $!{Test} ",
			replace: func(value string) (interface{}, error) {
				if value != "Test" {
					return "", errors.New("unexpected var name")
				}
				return "test", nil
			},
			output: " test abc test ",
		},
		"Single Escape": {
			input:   " test abc $${Test} ",
			replace: func(value string) (interface{}, error) { return "", errors.New("Shouldn't match at all") },
			output:  " test abc ${Test} ",
		},
		"Single Escape 2": {
			input:   " test abc $$${Test} ",
			replace: func(value string) (interface{}, error) { return "", errors.New("Shouldn't match at all") },
			output:  " test abc $${Test} ",
		},
		"Multiple Replace": {
			input:   " test ${ABC}${Test} abc $${Test}${Test} ",
			replace: func(value string) (interface{}, error) { return "test", nil },
			output:  " test testtest abc ${Test}test ",
		},
		"Multiple Replace 2": {
			input:   "${Test}${Test}${Test}",
			replace: func(value string) (interface{}, error) { return value, nil },
			output:  "TestTestTest",
		},
		"Return integer": {
			input:   "${integer}",
			replace: func(value string) (interface{}, error) { return "1", nil },
			output:  "1",
		},
		"Return bool": {
			input:   "${bool}",
			replace: func(value string) (interface{}, error) { return "true", nil },
			output:  "true",
		},
		"Return error": {
			input:   "${bool}",
			replace: func(value string) (interface{}, error) { return "", errors.New("Test Error") },
			err:     ptr.String("Test Error"),
		},
		"No match": {
			input:   "Test",
			replace: func(value string) (interface{}, error) { return "", errors.New("Test Error") },
			output:  "Test",
		},
		"Force String": {
			input:   "$!{Test}",
			replace: func(value string) (interface{}, error) { return 123, nil },
			output:  "123",
		},
		"Force String 2": {
			input:   "$!{Test}",
			replace: func(value string) (interface{}, error) { return "123", nil },
			output:  "123",
		},
		"Force String 3": {
			input:   "${Test}",
			replace: func(value string) (interface{}, error) { return "123", nil },
			output:  "123",
		},
	}

	// Run test cases
	for key, value := range testCases {
		out, err := ParseString(value.input, value.replace)
		if err != nil {
			if value.err != nil {
				if *value.err != err.Error() {
					t.Fatalf("Test %s failed: unexpected error message: expected %s - got %s", key, *value.err, err.Error())
				}
			} else {
				t.Fatalf("Test %s failed: %v", key, err)
			}
		} else if out != value.output {
			t.Fatalf("Test %s failed: expected %v - got %v", key, value.output, out)
		}
	}
}
