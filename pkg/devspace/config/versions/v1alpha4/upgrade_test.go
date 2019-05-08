package v1alpha4

import (
	"testing"

	next "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/v1beta1"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
)

func TestSimple(t *testing.T) {
	oldConfig := &Config{
		Dev: &DevConfig{
			Ports: &[]*PortForwardingConfig{
				{
					PortMappings: &[]*PortMapping{
						{
							LocalPort:   ptr.Int(3000),
							RemotePort:  ptr.Int(4000),
							BindAddress: nil,
						},
						{
							LocalPort:   ptr.Int(1000),
							RemotePort:  ptr.Int(1000),
							BindAddress: ptr.String("127.0.0.1"),
						},
					},
				},
				{
					LabelSelector: &map[string]*string{},
				},
				{
					PortMappings: &[]*PortMapping{
						{
							LocalPort:   ptr.Int(3000),
							RemotePort:  ptr.Int(4000),
							BindAddress: nil,
						},
					},
				},
			},
		},
	}

	newConfig, err := oldConfig.Upgrade()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	if newConfig == nil {
		t.Fatal("NewConfig is nil")
	}

	newConfigConverted, ok := newConfig.(*next.Config)
	if ok == false {
		t.Fatalf("Config couldn't get converted to version %s", next.Version)
	}
	if len(*newConfigConverted.Dev.Ports) != 3 {
		t.Fatalf("Error converting ports: %#v", *newConfigConverted.Dev.Ports)
	}
	if (*newConfigConverted.Dev.Ports)[0].PortMappings == nil || len(*(*newConfigConverted.Dev.Ports)[0].PortMappings) != 2 {
		t.Fatal("Wrong amount of port mappings in first port config")
	}
	if (*newConfigConverted.Dev.Ports)[1].PortMappings != nil {
		t.Fatal("Wrong amount of port mappings in second port config")
	}
	if (*newConfigConverted.Dev.Ports)[2].PortMappings == nil || len(*(*newConfigConverted.Dev.Ports)[2].PortMappings) != 1 {
		t.Fatal("Wrong amount of port mappings in third port config")
	}
}
