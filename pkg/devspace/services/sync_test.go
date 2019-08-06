package services

import (
	"testing"
)

func TestDownloadSyncHelper(t *testing.T) {
	t.Skip("Currently not working")

	err := downloadFile("does-not-exist", "does-not-exist")
	if err == nil || err.Error() != "Couldn't find sync helper in github release does-not-exist at url https://github.com/devspace-cloud/devspace/releases/tag/does-not-exist" {
		t.Fatalf("Unexpected error: %v", err)
	}
}
