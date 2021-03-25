package versions

import (
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1alpha1"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/ptr"

	"gotest.tools/assert"
)

func TestParse(t *testing.T) {
	config, err := Parse(map[interface{}]interface{}{
		"version": "DoesNotExist",
	}, log.Discard)
	assert.Error(t, err, "Unrecognized config version DoesNotExist. Please upgrade devspace with `devspace upgrade`")
	assert.Equal(t, true, config == nil, "Config from invalid version not nil")

	config, err = Parse(map[interface{}]interface{}{
		"version": latest.Version,
		"images": &map[string]*latest.ImageConfig{
			"TestImg": &latest.ImageConfig{
				Image: "TestImg",
			},
		},
	}, log.Discard)
	assert.NilError(t, err, "Error parsing map without defined version: %v")
	assert.Equal(t, latest.Version, config.Version, "Conversion to latest version not correct")
	assert.Equal(t, "TestImg", config.Images["TestImg"].Image, "Conversion to latest version not correct")

	config, err = Parse(map[interface{}]interface{}{
		"version": "v1alpha1",
		"images": &map[string]*v1alpha1.ImageConfig{
			"TestImg": &v1alpha1.ImageConfig{
				Name: ptr.String("TestImg"),
			},
		},
	}, log.Discard)
	assert.NilError(t, err, "Error parsing map with defined version v1alpha1: %v")
	assert.Equal(t, latest.Version, config.Version, "Conversion to latest version not correct")
	assert.Equal(t, "TestImg", config.Images["TestImg"].Image, "Conversion to latest version not correct")
}
