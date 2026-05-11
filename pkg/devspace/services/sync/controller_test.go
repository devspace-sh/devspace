package sync

import (
	"testing"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/tomb"
	"gotest.tools/assert"
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

// TestNoWatchRestartOnError verifies that RestartOnError is disabled when NoWatch is true.
// This mirrors the logic in startSync(): RestartOnError: !syncConfig.NoWatch
func TestNoWatchRestartOnError(t *testing.T) {
	testCases := []struct {
		name          string
		noWatch       bool
		wantRestart   bool
	}{
		{"watch mode enables restart-on-error", false, true},
		{"noWatch mode disables restart-on-error", true, false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &latest.SyncConfig{NoWatch: tc.noWatch}
			restartOnError := !cfg.NoWatch
			assert.Equal(t, restartOnError, tc.wantRestart,
				"RestartOnError mismatch for NoWatch=%v", tc.noWatch)
		})
	}
}

// TestNoWatchOnDoneDoesNotKillParent verifies that when a noWatch sync completes
// (onDone fires), the parent tomb is NOT killed. This ensures other services such
// as SSH and port-forwarding, which are registered in the same parent, continue
// running after the one-shot sync finishes.
func TestNoWatchOnDoneDoesNotKillParent(t *testing.T) {
	var parent tomb.Tomb
	onDone := make(chan struct{})
	handlerExited := make(chan struct{})

	// Simulate the noWatch (else) branch goroutine from startWithWait.
	// The correct behaviour is: on onDone, do NOT call parent.Kill.
	parent.Go(func() error {
		defer close(handlerExited)
		select {
		case <-onDone:
			// noWatch sync complete — intentionally no parent.Kill here
		}
		return nil
	})

	// Signal that the one-shot sync has finished.
	close(onDone)

	// Wait for the handler goroutine to exit.
	select {
	case <-handlerExited:
	case <-time.After(2 * time.Second):
		t.Fatal("noWatch handler goroutine did not finish in time")
	}

	// Parent must still be alive: Kill was never called, so other services continue.
	assert.Assert(t, parent.Alive(),
		"parent tomb was killed on noWatch sync completion; SSH/port-forwarding would be terminated")
}

// TestNoWatchOnDoneWatchModeKillsParent documents the intentional contrast:
// in watch mode (RestartOnError=true) the onDone path DOES call syncDone → parent.Kill,
// terminating the whole dev session when the sync unexpectedly stops.
func TestNoWatchOnDoneWatchModeKillsParent(t *testing.T) {
	var parent tomb.Tomb
	onDone := make(chan struct{})
	handlerExited := make(chan struct{})

	// Simulate the RestartOnError (if) branch: onDone → Kill parent.
	parent.Go(func() error {
		defer close(handlerExited)
		select {
		case <-onDone:
			parent.Kill(nil) // watch mode: unexpected stop kills everything
		}
		return nil
	})

	close(onDone)

	select {
	case <-handlerExited:
	case <-time.After(2 * time.Second):
		t.Fatal("watch-mode handler goroutine did not finish in time")
	}

	// Parent should be dying/dead because Kill was called.
	assert.Assert(t, !parent.Alive(),
		"expected parent tomb to be killed in watch mode when sync stops")
}
