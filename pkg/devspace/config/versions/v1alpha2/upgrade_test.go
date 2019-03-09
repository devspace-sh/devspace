package v1alpha2

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/util/ptr"
)

func TestSimple(t *testing.T) {
	oldConfig := &Config{
		Cluster: &Cluster{
			CloudProvider: ptr.String("test"),
		},
	}

	newConfig, err := oldConfig.Upgrade()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	if newConfig == nil {
		t.Fatal("NewConfig is nil")
	}
}
