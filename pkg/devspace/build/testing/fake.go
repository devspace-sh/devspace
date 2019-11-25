package testing

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/build"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/randutil"
)

// FakeController is the fake build controller
type FakeController struct {
	BuiltImages map[string]string
}

// NewFakeController creates a new fake build controller
func NewFakeController(config *latest.Config) build.Controller {
	builtImages := map[string]string{}
	for _, imageConf := range config.Images {
		if imageConf.Build != nil && imageConf.Build.Disabled != nil && *imageConf.Build.Disabled == true {
			continue
		}

		// This is necessary for parallel build otherwise we would override the image conf pointer during the loop
		cImageConf := *imageConf
		imageName := cImageConf.Image

		// Get image tag
		imageTag, _ := randutil.GenerateRandomString(7)
		if imageConf.Tag != "" {
			imageTag = imageConf.Tag
		}

		builtImages[imageName] = imageTag
	}

	return &FakeController{
		BuiltImages: builtImages,
	}
}

// Build builds the images
func (f *FakeController) Build(options *build.Options, log log.Logger) (map[string]string, error) {
	return f.BuiltImages, nil
}
