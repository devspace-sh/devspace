package util

import (
	"errors"
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

func TestRunDownloadOnceReturnsErrorToConcurrentWaiter(t *testing.T) {
	key := "test-download-call-error-waiter"
	expectedErr := errors.New("download failed")
	call := &downloadCall{done: make(chan struct{})}

	downloadCallsMutex.Lock()
	downloadCalls[key] = call
	downloadCallsMutex.Unlock()
	defer func() {
		downloadCallsMutex.Lock()
		delete(downloadCalls, key)
		downloadCallsMutex.Unlock()
	}()

	errs := make(chan error, 1)
	go func() {
		errs <- runDownloadOnce(key, func() error {
			return nil
		})
	}()

	call.err = expectedErr
	close(call.done)

	assert.Equal(t, <-errs, expectedErr)
}

func TestRunDownloadOnceCleansUpAfterPanic(t *testing.T) {
	key := "test-download-call-panic-cleanup"

	func() {
		defer func() {
			recovered := recover()
			assert.Assert(t, recovered != nil, "expected panic")
		}()

		_ = runDownloadOnce(key, func() error {
			panic("download panic")
		})
	}()

	downloadCallsMutex.Lock()
	_, ok := downloadCalls[key]
	downloadCallsMutex.Unlock()
	assert.Assert(t, !ok, "expected download call to be removed after panic")
}
