package loader

import (
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"gotest.tools/assert"
)

func TestValidateImageName(t *testing.T) {
	config := &latest.Config{
		Images: map[string]*latest.ImageConfig{
			"default": {
				Image: "localhost:5000/node",
			},
		},
	}
	err := validateImages(config)
	assert.NilError(t, err)

	config = &latest.Config{
		Images: map[string]*latest.ImageConfig{
			"default": {
				Image: "localhost:5000/node:latest",
			},
		},
	}
	err = validateImages(config)
	assert.Error(t, err, "images.default.image 'localhost:5000/node:latest' can not have tag 'latest'")
}
