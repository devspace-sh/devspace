package sync

import (
	"gotest.tools/assert"
	"testing"
)

type parseSyncPathTestCase struct {
	name string
	in   string

	expectedLocal  string
	expectedRemote string
}

func TestParseSyncPath(t *testing.T) {
	testCases := []parseSyncPathTestCase{
		{
			name:           "Test Windows",
			in:             "C:/codeproject:/home/dev/codeproject",
			expectedLocal:  "C:/codeproject",
			expectedRemote: "/home/dev/codeproject",
		},
	}

	for _, testCase := range testCases {
		local, remote, err := ParseSyncPath(testCase.in)
		assert.NilError(t, err)
		assert.Equal(t, local, testCase.expectedLocal, "Expect local path in "+testCase.name)
		assert.Equal(t, remote, testCase.expectedRemote, "Expect remote path in "+testCase.name)
	}
}

func TestInitialSyncCompletedMessage(t *testing.T) {
	testCases := []struct {
		name     string
		in       string
		expected string
	}{
		{
			name:     "empty path defaults to dot",
			in:       "",
			expected: "Initial sync completed for . <-> .",
		},
		{
			name:     "relative directory pair",
			in:       "../foo:/src/projects/foo",
			expected: "Initial sync completed for ../foo <-> /src/projects/foo",
		},
		{
			name:     "single path matching issue 3309",
			in:       "/projects/qux",
			expected: "Initial sync completed for /projects/qux <-> /projects/qux",
		},
		{
			name:     "windows path with colon",
			in:       "C:/codeproject:/home/dev/codeproject",
			expected: "Initial sync completed for C:/codeproject <-> /home/dev/codeproject",
		},
	}

	for _, tc := range testCases {
		msg := InitialSyncCompletedMessage(tc.in)
		assert.Equal(t, msg, tc.expected, tc.name)
	}
}
