package v1alpha1

import (
	"testing"

	next "github.com/covexo/devspace/pkg/devspace/config/versions/latest"
	"github.com/covexo/devspace/pkg/util/ptr"
)

func TestEmpty(t *testing.T) {
	oldConfig := New()
	oldConfigConverted := oldConfig.(*Config)

	newConfig, err := oldConfigConverted.Upgrade()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	if newConfig == nil {
		t.Fatal("NewConfig is nil")
	}
}

func TestSimple(t *testing.T) {
	oldConfig := &Config{
		DevSpace: &DevSpaceConfig{
			Deployments: &[]*DeploymentConfig{
				&DeploymentConfig{
					Name: ptr.String("test"),
					Helm: &HelmConfig{
						DevOverwrite: ptr.String("overwrite"),
					},
				},
			},
			Services: &[]*ServiceConfig{
				{
					Name:      ptr.String("test"),
					Namespace: ptr.String("testnamespace"),
				},
			},
			Ports: &[]*PortForwardingConfig{
				{
					Service: ptr.String("test"),
				},
			},
		},
		Images: &map[string]*ImageConfig{
			"test": &ImageConfig{
				Name:     ptr.String("test"),
				Registry: ptr.String("test"),
			},
		},
		Registries: &map[string]*RegistryConfig{
			"test": &RegistryConfig{
				URL: ptr.String("test.io"),
			},
		},
		Tiller: &TillerConfig{
			Namespace: ptr.String("tillernamespace"),
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
	if (*newConfigConverted.DevSpace.Deployments)[0].Helm.Overrides == nil || len(*(*newConfigConverted.DevSpace.Deployments)[0].Helm.Overrides) == 0 || *(*(*newConfigConverted.DevSpace.Deployments)[0].Helm.Overrides)[0] != "overwrite" {
		t.Fatal("Error converting devOverwrite")
	}
	if (*newConfigConverted.DevSpace.Deployments)[0].Helm.TillerNamespace == nil || *(*newConfigConverted.DevSpace.Deployments)[0].Helm.TillerNamespace != "tillernamespace" {
		t.Fatal("Error converting tiller namespace")
	}

	// Check selectors
	if newConfigConverted.DevSpace.Selectors == nil || len(*newConfigConverted.DevSpace.Selectors) != 1 {
		t.Fatal("Error converting services")
	}
	if newConfigConverted.DevSpace.Ports == nil || len(*newConfigConverted.DevSpace.Ports) != 1 {
		t.Fatal("Error converting ports")
	}
	if *(*newConfigConverted.DevSpace.Selectors)[0].Name != "test" || *(*newConfigConverted.DevSpace.Ports)[0].Selector != "test" {
		t.Fatal("Error converting service")
	}

	// Check image registries
	if newConfigConverted.Images == nil || (*newConfigConverted.Images)["test"] == nil {
		t.Fatal("Error converting images")
	}
	if *(*newConfigConverted.Images)["test"].Name != "test.io/test" {
		t.Fatal("Error converting image name")
	}
}
