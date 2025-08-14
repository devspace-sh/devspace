package commands

import (
	"bytes"
	"os"
	"runtime"
	"strings"
	"testing"

	"gotest.tools/assert"
	"mvdan.cc/sh/v3/interp"
)

// this test implies to cat testFile
func TestCat(t *testing.T) {
	f, err := os.CreateTemp(".", "testFile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	if _, err = f.WriteString("Hello DevSpace!"); err != nil {
		t.Fatalf("Unable to write to temporary file %v", err)
	}

	expectedOutput := "Hello DevSpace!"
	stdout := &bytes.Buffer{}
	stdin := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx := &interp.HandlerContext{
		Dir:    ".",
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	}
	args := []string{
		f.Name(),
	}

	assert.Equal(t, stdout.String(), "")

	err = Cat(ctx, args)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, stderr.String(), "")
	assert.Equal(t, stdout.String(), expectedOutput)
}

// this test implies to echo Hello DevSpace! | cat
func TestCatNoArgs(t *testing.T) {
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("Hello DevSpace!")
	stderr := &bytes.Buffer{}
	ctx := &interp.HandlerContext{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	}
	args := []string{}
	expectedOutput := "Hello DevSpace!"

	assert.Equal(t, stdout.String(), "")

	err := Cat(ctx, args)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, stderr.String(), "")
	assert.Equal(t, stdout.String(), expectedOutput)
}

func TestCatNoFile(t *testing.T) {
	expectedOutput := func() string {
		if runtime.GOOS == "windows" {
			return "cat: open randomFile.txt: The system cannot find the file specified."
		}
		return "cat: open randomFile.txt: no such file or directory"
	}()

	stdout := &bytes.Buffer{}
	stdin := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx := &interp.HandlerContext{
		Dir:    ".",
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	}
	args := []string{"randomFile.txt"}

	err := Cat(ctx, args)
	assert.Error(t, err, expectedOutput)
}
