package v1alpha3

import (
	"testing"

	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/v1alpha4"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/ptr"
)

func TestSimple(t *testing.T) {
	oldConfig := &Config{
		Deployments: &[]*DeploymentConfig{
			{
				Name: ptr.String("test-deployment"),
			},
			{
				Name: ptr.String("test-deployment-helm"),
				Helm: &HelmConfig{
					ChartPath: ptr.String("chart/"),
					Overrides: &[]*string{
						ptr.String("chart/values.yaml"),
					},
					OverrideValues: &map[interface{}]interface{}{
						"test": "test",
					},
				},
			},
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
	if len(*newConfigConverted.Deployments) != 2 {
		t.Fatalf("Error converting deployments: %#v", *newConfigConverted.Deployments)
	}
	if *(*newConfigConverted.Deployments)[0].Name != "test-deployment" || *(*newConfigConverted.Deployments)[1].Name != "test-deployment-helm" {
		t.Fatal("Wrong order of deployments")
	}
	if len(*(*newConfigConverted.Deployments)[1].Helm.ValuesFiles) != 1 || *(*(*newConfigConverted.Deployments)[1].Helm.ValuesFiles)[0] != "chart/values.yaml" {
		t.Fatalf("Helm.Overrides was not correctly converted: %#v", *(*newConfigConverted.Deployments)[1].Helm.ValuesFiles)
	}
	if (*newConfigConverted.Deployments)[1].Helm.Chart == nil || (*newConfigConverted.Deployments)[1].Helm.Chart.Name == nil || *(*newConfigConverted.Deployments)[1].Helm.Chart.Name != "chart/" {
		t.Fatalf("Helm.ChartPath was not correctly converted")
	}
	if (*newConfigConverted.Deployments)[1].Helm.Values == nil || (*(*newConfigConverted.Deployments)[1].Helm.Values)["test"] != "test" {
		t.Fatalf("Helm.OverrideValues was not correctly converted")
	}
}
