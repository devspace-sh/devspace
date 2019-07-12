package latest

import (
	"testing"

	"gotest.tools/assert"
)

func TestNewUpgrade(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("No panic when trying to upgrade latest")
		}
	}()
	New().Upgrade()
}

func TestNew(t *testing.T) {
	config := New()
	assert.Equal(t, config.GetVersion(), Version, "Wrong version of new Config")
}

func TestNewRaw(t *testing.T) {
	config := NewRaw()
	assert.Equal(t, config.GetVersion(), Version, "Wrong version of new Config")
	assert.Equal(t, len(*config.Images), 0, "Config initialized with images")
}
