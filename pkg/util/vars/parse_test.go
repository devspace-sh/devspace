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
		"Single Replace": &testCase{
			input:   " test abc ${Test} ",
			replace: func(value string) (interface{}, bool, error) { return "test", false, nil },
			output:  " test abc test ",
		},
		"Var Name 2": &testCase{
			input: " test abc ${Test} ",
			replace: func(value string) (interface{}, bool, error) {
				if value != "Test" {
					return "", false, errors.New("unexpected var name")
				}
				return "test", false, nil
			},
			output: " test abc test ",
		},
		"Var Name": &testCase{
			input: " test abc $!{Test} ",
			replace: func(value string) (interface{}, bool, error) {
				if value != "Test" {
					return "", false, errors.New("unexpected var name")
				}
				return "test", false, nil
			},
			output: " test abc test ",
		},
		"Single Escape": &testCase{
			input:   " test abc $${Test} ",
			replace: func(value string) (interface{}, bool, error) { return "", false, errors.New("Shouldn't match at all") },
			output:  " test abc ${Test} ",
		},
		"Single Escape 2": &testCase{
			input:   " test abc $$${Test} ",
			replace: func(value string) (interface{}, bool, error) { return "", false, errors.New("Shouldn't match at all") },
			output:  " test abc $${Test} ",
		},
		"Multiple Replace": &testCase{
			input:   " test ${ABC}${Test} abc $${Test}${Test} ",
			replace: func(value string) (interface{}, bool, error) { return "test", false, nil },
			output:  " test testtest abc ${Test}test ",
		},
		"Multiple Replace 2": &testCase{
			input:   "${Test}${Test}${Test}",
			replace: func(value string) (interface{}, bool, error) { return value, false, nil },
			output:  "TestTestTest",
		},
		"Return integer": &testCase{
			input:   "${integer}",
			replace: func(value string) (interface{}, bool, error) { return "1", false, nil },
			output:  1,
		},
		"Return bool": &testCase{
			input:   "${bool}",
			replace: func(value string) (interface{}, bool, error) { return "true", false, nil },
			output:  true,
		},
		"Return error": &testCase{
			input:   "${bool}",
			replace: func(value string) (interface{}, bool, error) { return "", false, errors.New("Test Error") },
			err:     ptr.String("Test Error"),
		},
		"No match": &testCase{
			input:   "Test",
			replace: func(value string) (interface{}, bool, error) { return "", false, errors.New("Test Error") },
			output:  "Test",
		},
		"Force String": &testCase{
			input:   "$!{Test}",
			replace: func(value string) (interface{}, bool, error) { return 123, false, nil },
			output:  "123",
		},
		"Force String 2": &testCase{
			input:   "$!{Test}",
			replace: func(value string) (interface{}, bool, error) { return "123", false, nil },
			output:  "123",
		},
		"Force String 3": &testCase{
			input:   "${Test}",
			replace: func(value string) (interface{}, bool, error) { return "123", true, nil },
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
