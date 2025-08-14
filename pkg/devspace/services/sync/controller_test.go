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
