package shell

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"gotest.tools/assert"
	"mvdan.cc/sh/v3/expand"
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
			command:        "cat " + file.Name(),
			expectedOutput: "Hello DevSpace!",
		},
		{
			command:        "echo 123 | cat",
			expectedOutput: "123\n",
		},
	}

	for _, testCase := range testCases {
		stdout := &bytes.Buffer{}
		err := ExecuteShellCommand(testCase.command, nil, ".", stdout, nil, nil)
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
		err := ExecuteShellCommand(testCase.command, nil, ".", nil, stderr, nil)
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
			command:        "cat " + file.Name(),
			expectedOutput: "Hello DevSpace!",
		},
		{
			command:        "echo 123 | cat",
			expectedOutput: "123\n",
		},
	}
	lookPathDir = func(cwd string, env expand.Environ, file string) (string, error) {
		return "", errors.New("not found")
	}
	for _, testCase := range testCases {
		stdout := &bytes.Buffer{}
		err := ExecuteShellCommand(testCase.command, nil, ".", stdout, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, stdout.String(), testCase.expectedOutput)
	}
}

func TestKubectlDownload(t *testing.T) {
	lookPathDir = func(cwd string, env expand.Environ, file string) (string, error) {
		return "", errors.New("not found")
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := ExecuteShellCommand("kubectl", nil, ".", stdout, stderr, nil)
	if err != nil {
		t.Fatal(err)
	}
	stdout1 := &bytes.Buffer{}
	err = ExecuteShellCommand("kubectl version", nil, ".", stdout1, stderr, nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.Assert(t, strings.Contains(stdout1.String(), `Client Version`))
}

func TestHelmDownload(t *testing.T) {
	lookPathDir = func(cwd string, env expand.Environ, file string) (string, error) {
		return "", errors.New("not found")
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := ExecuteShellCommand("helm", nil, ".", stdout, stderr, nil)
	if err != nil {
		t.Fatal(err)
	}
	stdout1 := &bytes.Buffer{}
	err = ExecuteShellCommand("helm version", nil, ".", stdout1, stderr, nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.Assert(t, strings.Contains(stdout1.String(), `Version:"v3`))
}
