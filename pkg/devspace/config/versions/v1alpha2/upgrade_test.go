package v1alpha2

import (
	"testing"

	next "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
)

func TestSimple(t *testing.T) {
	oldConfig := &Config{
		Cluster: &Cluster{
			CloudProvider: ptr.String("test"),
		},
		Images: &map[string]*ImageConfig{
			"default": &ImageConfig{
				Image: ptr.String("test"),
				Build: &BuildConfig{
					DockerfilePath: ptr.String("mydockerfile"),
					ContextPath:    ptr.String("mycontextpath"),
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
	if newConfigConverted.Images == nil || len(*newConfigConverted.Images) != 1 {
		t.Fatal("Converting images was not successful")
	}
	if (*newConfigConverted.Images)["default"] == nil || (*newConfigConverted.Images)["default"].Build == nil || (*newConfigConverted.Images)["default"].Build.Dockerfile == nil || (*newConfigConverted.Images)["default"].Build.Context == nil {
		t.Fatal("Converting image default was not successful")
	}
	if *(*newConfigConverted.Images)["default"].Build.Dockerfile != "mydockerfile" || *(*newConfigConverted.Images)["default"].Build.Context != "mycontextpath" {
		t.Fatal("Wrong values for dockerfilepath or contextpath")
	}
}
