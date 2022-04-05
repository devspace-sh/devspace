package testing

import (
	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/util/randutil"
)

// FakeController is the fake build controller
type FakeController struct {
	BuiltImages map[string]string
}

// NewFakeController creates a new fake build controller
func NewFakeController(config *latest.Config) build.Controller {
	builtImages := map[string]string{}
	for _, image := range config.Images {
		if image != nil && image.Docker == nil && image.Kaniko == nil && image.BuildKit == nil && image.Custom == nil {
			continue
		}

		// This is necessary for parallel build otherwise we would override the image conf pointer during the loop
		cImageConf := *image
		imageName := cImageConf.Image

		// Get image tag
		imageTag := randutil.GenerateRandomString(7)
		if len(image.Tags) > 0 {
			imageTag = image.Tags[0]
		}

		builtImages[imageName] = imageTag
	}

	return &FakeController{
		BuiltImages: builtImages,
	}
}

// Build builds the images
func (f *FakeController) Build(ctx devspacecontext.Context, images []string, options *build.Options) error {
	return nil
}
