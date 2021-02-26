package v1alpha1

import (
	"testing"

	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/v1alpha2"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/ptr"
)

func TestEmpty(t *testing.T) {
	oldConfig := New()
	oldConfigConverted := oldConfig.(*Config)

	newConfig, err := oldConfigConverted.Upgrade(log.Discard)
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
			Sync: &[]*SyncConfig{
				{
					Namespace: ptr.String("test"),
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

	newConfig, err := oldConfig.Upgrade(log.Discard)
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
	if (*newConfigConverted.Deployments)[0].Helm.Overrides == nil || len(*(*newConfigConverted.Deployments)[0].Helm.Overrides) == 0 || *(*(*newConfigConverted.Deployments)[0].Helm.Overrides)[0] != "overwrite" {
		t.Fatal("Error converting devOverwrite")
	}
	if (*newConfigConverted.Deployments)[0].Helm.TillerNamespace == nil || *(*newConfigConverted.Deployments)[0].Helm.TillerNamespace != "tillernamespace" {
		t.Fatal("Error converting tiller namespace")
	}

	// Check selectors
	if newConfigConverted.Dev.Selectors == nil || len(*newConfigConverted.Dev.Selectors) != 1 {
		t.Fatal("Error converting services")
	}
	if newConfigConverted.Dev.Ports == nil || len(*newConfigConverted.Dev.Ports) != 1 {
		t.Fatal("Error converting ports")
	}
	if *(*newConfigConverted.Dev.Selectors)[0].Name != "test" || *(*newConfigConverted.Dev.Ports)[0].Selector != "test" {
		t.Fatal("Error converting service")
	}

	// Check image registries
	if newConfigConverted.Images == nil || (*newConfigConverted.Images)["test"] == nil {
		t.Fatal("Error converting images")
	}
	if *(*newConfigConverted.Images)["test"].Image != "test.io/test" {
		t.Fatal("Error converting image name")
	}
}
