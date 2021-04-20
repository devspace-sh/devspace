package command

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

func TestCommand(t *testing.T) {
	p := syntax.NewParser()

	file, err := p.Parse(strings.NewReader("printf 123 && printf '4'\"'\"'5''6'"), "")
	if err != nil {
		t.Fatal(err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	r, err := interp.New(interp.StdIO(nil, stdout, stderr))
	if err != nil {
		t.Fatal(err)
	}

	err = r.Run(context.Background(), file)
	if err != nil {
		if status, ok := interp.IsExitStatus(err); ok {
			if status != 0 {
				t.Fatal(err)
			}
		} else {
			t.Fatal(err)
		}
	}

	if stdout.String() != "1234'56" {
		t.Fatalf("Expected stdout '1234'56', got stdout '%s' - stderr '%s'", stdout.String(), stderr.String())
	}
}
