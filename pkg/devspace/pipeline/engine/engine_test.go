package engine

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/assert"
)

type testCaseShell struct {
	command        string
	expectedOutput string
}

func TestShellCat(t *testing.T) {
	file, err := ioutil.TempFile(".", "testFile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	if _, err = file.WriteString("Hello DevSpace!"); err != nil {
		t.Fatalf("Unable to write to temporary file %v", err)
	}

	testCases := []testCaseShell{
		{
			command:        "cat " + filepath.ToSlash(file.Name()),
			expectedOutput: "Hello DevSpace!",
		},
		{
			command:        "echo 123 | cat",
			expectedOutput: "123\n",
		},
	}

	for _, testCase := range testCases {
		stdout := &bytes.Buffer{}
		err := ExecuteSimpleShellCommand(context.Background(), ".", stdout, nil, nil, nil, testCase.command)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, stdout.String(), testCase.expectedOutput)
	}
}

func TestShellCatError(t *testing.T) {
	testCases := []testCaseShell{
		{
			command:        "cat noFile.txt",
			expectedOutput: "cat: noFile.txt: No such file or directory\n",
		},
	}

	for _, testCase := range testCases {
		stderr := &bytes.Buffer{}
		err := ExecuteSimpleShellCommand(context.Background(), ".", nil, stderr, nil, nil, testCase.command)
		if err != nil {
			if stderr.String() != "" {
				assert.Equal(t, stderr.String(), testCase.expectedOutput)
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal("FAIL: TestShellCatError")
		}

	}
}

// this test forces the cat implementation to execute
func TestShellCatEnforce(t *testing.T) {
	file, err := ioutil.TempFile(".", "testFile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	if _, err = file.WriteString("Hello DevSpace!"); err != nil {
		t.Fatalf("Unable to write to temporary file %v", err)
	}

	testCases := []testCaseShell{
		{
			command:        "cat " + filepath.ToSlash(file.Name()),
			expectedOutput: "Hello DevSpace!",
		},
		{
			command:        "echo 123 | cat",
			expectedOutput: "123\n",
		},
	}

	for _, testCase := range testCases {
		stdout := &bytes.Buffer{}
		err := ExecuteSimpleShellCommand(context.Background(), ".", stdout, nil, nil, nil, testCase.command)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, stdout.String(), testCase.expectedOutput)
	}
}

func TestKubectlDownload(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := ExecuteSimpleShellCommand(context.Background(), ".", stdout, stderr, nil, nil, "kubectl")
	if err != nil {
		t.Fatal(err)
	}
	stdout1 := &bytes.Buffer{}
	err = ExecuteSimpleShellCommand(context.Background(), ".", stdout1, stderr, nil, nil, "kubectl -h")
	if err != nil {
		t.Fatal(err)
	}
}

func TestHelmDownload(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := ExecuteSimpleShellCommand(context.Background(), ".", stdout, stderr, nil, nil, "helm")
	if err != nil {
		t.Fatal(err)
	}
	stdout1 := &bytes.Buffer{}
	err = ExecuteSimpleShellCommand(context.Background(), ".", stdout1, stderr, nil, nil, "helm version")
	if err != nil {
		t.Fatal(err)
	}
	assert.Assert(t, strings.Contains(stdout1.String(), `Version:"v3`))
}
