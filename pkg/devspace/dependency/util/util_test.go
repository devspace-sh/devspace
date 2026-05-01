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

func TestDownloadLockCleansUp(t *testing.T) {
	id := "test-lock-cleanup"
	err := withDownloadLock(id, func() error {
		return nil
	})
	assert.NilError(t, err)

	downloadLocksMutex.Lock()
	_, ok := downloadLocks[id]
	downloadLocksMutex.Unlock()
	assert.Assert(t, !ok, "expected download lock to be removed")
}

func TestRunDownloadOnceCleansUpCall(t *testing.T) {
	key := "test-download-call-cleanup"
	err := runDownloadOnce(key, func() error {
		return nil
	})
	assert.NilError(t, err)

	downloadCallsMutex.Lock()
	_, ok := downloadCalls[key]
	downloadCallsMutex.Unlock()
	assert.Assert(t, !ok, "expected download call to be removed")
}
