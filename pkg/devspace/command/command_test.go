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

	file, err := p.Parse(strings.NewReader("go version && echo 123"), "")
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
	if err != nil && err != interp.ShellExitStatus(0) {
		t.Fatal(err)
	}

	t.Fatalf("Done - stdout '%s' - stderr '%s'", stdout.String(), stderr.String())
}
