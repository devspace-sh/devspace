package downloader

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
)

func TestHelmCommandIsValid(t *testing.T) {
	// Create a dummy binary to simulate helm output
	tmpDir := t.TempDir()
	dummyBin := filepath.Join(tmpDir, "dummy-helm")
	if runtime.GOOS == "windows" {
		dummyBin += ".exe"
	}

	// Because we can't easily compile a mock binary in a test without overhead,
	// DevSpace tests typically just rely on structural logic or mock interfaces.
	// But we can instantiate our command and ensure it doesn't panic on empty input.
	cmd := NewHelmCommand()
	if cmd.Name() != "helm" {
		t.Errorf("expected helm, got %s", cmd.Name())
	}

	// This should fail gracefully because dummyBin doesn't exist
	valid, err := cmd.IsValid(context.Background(), dummyBin)
	if err != nil {
		t.Errorf("expected no error from missing binary, got %v", err)
	}
	if valid {
		t.Error("expected missing binary to be invalid")
	}
}
