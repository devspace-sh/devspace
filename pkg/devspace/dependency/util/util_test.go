package util

import (
	"testing"

	"gotest.tools/assert"
)

func TestSwitchURLType(t *testing.T) {
	httpURL := "https://github.com/devspace-sh/devspace.git"
	sshURL := "git@github.com:devspace-sh/devspace.git"

	assert.Equal(t, sshURL, switchURLType(httpURL))
	assert.Equal(t, httpURL, switchURLType(sshURL))
}
