package upgrade

import (
	"testing"

	"gotest.tools/assert"
)

func TestSetVersion(t *testing.T) {
	SetVersion("sasd0.0.1hello")
	assert.Equal(t, "0.0.1hello", GetVersion(), "Wrong version set")
}

func TestEraseVersionPrefix(t *testing.T) {
	prefixless, err := eraseVersionPrefix("sasd0.0.1hello")
	if err != nil {
		t.Fatalf("Error erasing Version: %v", err)
	}
	assert.Equal(t, "0.0.1hello", prefixless, "Wrong version set")

	_, err = eraseVersionPrefix(".0.1hello")
	assert.Equal(t, true, err != nil, "No error returned with invalid string")
}
