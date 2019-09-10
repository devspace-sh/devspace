package services

import (
	"testing"
	
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	
	"gotest.tools/assert"
)

func TestStartPortForwarding(t *testing.T) {
	config := &latest.Config{
		Dev : &latest.DevConfig{},
	}
	portForwarder, err := StartPortForwarding(config, nil, nil, &log.DiscardLogger{}) 
	if err != nil {
		t.Fatalf("Error starting port forwarding with nil ports to forward: %v", err)
	}
	assert.Equal(t, true, portForwarder == nil, "Portforwarder returned despite nil port given to forward.")

	config = &latest.Config{
		Dev : &latest.DevConfig{
			Ports: []*latest.PortForwardingConfig{},
		},
	}
	portForwarder, err = StartPortForwarding(config, nil, nil, &log.DiscardLogger{})
	if err != nil {
		t.Fatalf("Error starting port forwarding with 0 ports to forward: %v", err)
	}
	assert.Equal(t, 0, len(portForwarder), "Ports forwarded despite 0 ports given to forward.")
}
