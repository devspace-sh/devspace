package versions

import (
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	"gotest.tools/assert"
)

func TestParse(t *testing.T) {
	config, err := Parse(map[string]interface{}{
		"version": "DoesNotExist",
	}, log.Discard)
	assert.Error(t, err, "Unrecognized config version DoesNotExist. Please upgrade devspace with `devspace upgrade`")
	assert.Equal(t, true, config == nil, "Config from invalid version not nil")

	config, err = Parse(map[string]interface{}{
		"version": latest.Version,
		"images": &map[string]*latest.Image{
			"TestImg": {
				Image: "TestImg",
			},
		},
	}, log.Discard)
	assert.NilError(t, err, "Error parsing map without defined version: %v")
	assert.Equal(t, latest.Version, config.Version, "Conversion to latest version not correct")
	assert.Equal(t, "TestImg", config.Images["TestImg"].Image, "Conversion to latest version not correct")
}
