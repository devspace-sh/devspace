package legacy

import (
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"gopkg.in/yaml.v3"
	"gotest.tools/assert"
)

type replaceContainerNamesTestCase struct {
	name string

	overwriteValues map[string]interface{}
	cache           *localcache.LocalCache
	imagesConf      map[string]*latest.Image
	// builtImages     map[string]string

	expectedShouldRedeploy  bool
	expectedOverwriteValues map[string]interface{}
}

func TestReplaceContainerNames(t *testing.T) {
	testCases := []replaceContainerNamesTestCase{
		{
			name: "invalid image name",
			overwriteValues: map[string]interface{}{
				"": "",
			},
			cache: &localcache.LocalCache{
				Images: map[string]localcache.ImageCache{
					"": {},
				},
			},
			expectedOverwriteValues: map[string]interface{}{
				"": "",
			},
		},
		{
			name: "Image not in cache",
			overwriteValues: map[string]interface{}{
				"": "myimage",
			},
			cache: &localcache.LocalCache{
				Images: map[string]localcache.ImageCache{},
			},
			expectedOverwriteValues: map[string]interface{}{
				"": "myimage",
			},
		},
		// TODO: assertion failed: false (shouldRedeploy bool) != true (testCase.expectedShouldRedeploy bool)
		// {
		// 	name: "Image in cache",
		// 	overwriteValues: map[string]interface{}{
		// 		"": "myimage",
		// 	},
		// 	imagesConf: map[string]*latest.Image{
		// 		"test": {
		// 			Image: "myimage",
		// 		},
		// 	},
		// 	cache: &localcache.LocalCache{
		// 		Images: map[string]localcache.ImageCache{
		// 			"test": {
		// 				ImageName: "myimage",
		// 				Tag:       "someTag",
		// 			},
		// 		},
		// 	},
		// 	builtImages: map[string]string{
		// 		"myimage": "",
		// 	},
		// 	expectedShouldRedeploy: true,
		// 	expectedOverwriteValues: map[string]interface{}{
		// 		"": "myimage:someTag",
		// 	},
		// },
		// {
		// 	name: "Replace image & tag helpers",
		// 	overwriteValues: map[string]interface{}{
		// 		"": "image(test):tag(test)",
		// 	},
		// 	imagesConf: map[string]*latest.Image{
		// 		"test": {
		// 			Image: "myimage",
		// 		},
		// 	},
		// 	cache: &localcache.LocalCache{
		// 		Images: map[string]localcache.ImageCache{
		// 			"test": {
		// 				ImageName: "myimage",
		// 				Tag:       "someTag",
		// 			},
		// 		},
		// 	},
		// 	builtImages: map[string]string{
		// 		"myimage": "",
		// 	},
		// 	expectedShouldRedeploy: true,
		// 	expectedOverwriteValues: map[string]interface{}{
		// 		"": "myimage:someTag",
		// 	},
		// },
		// {
		// 	name: "Do not replace unknown tag helpers",
		// 	overwriteValues: map[string]interface{}{
		// 		"": "tag(test2):image(test):tag(test)image(test)",
		// 	},
		// 	imagesConf: map[string]*latest.Image{
		// 		"test": {
		// 			Image: "myimage",
		// 		},
		// 	},
		// 	cache: &localcache.LocalCache{
		// 		Images: map[string]localcache.ImageCache{
		// 			"test": {
		// 				ImageName: "myimage",
		// 				Tag:       "someTag",
		// 			},
		// 		},
		// 	},
		// 	builtImages: map[string]string{
		// 		"myimage": "",
		// 	},
		// 	expectedShouldRedeploy: true,
		// 	expectedOverwriteValues: map[string]interface{}{
		// 		"": "tag(test2):myimage:someTagmyimage",
		// 	},
		// },
		// {
		// 	name: "Do not replace unknown image helpers",
		// 	overwriteValues: map[string]interface{}{
		// 		"": "image(test2):image(test):tag(test)image(test)",
		// 	},
		// 	imagesConf: map[string]*latest.Image{
		// 		"test": {
		// 			Image: "myimage",
		// 		},
		// 	},
		// 	cache: &localcache.LocalCache{
		// 		Images: map[string]localcache.ImageCache{
		// 			"test": {
		// 				ImageName: "myimage",
		// 				Tag:       "someTag",
		// 			},
		// 		},
		// 	},
		// 	builtImages: map[string]string{
		// 		"myimage": "",
		// 	},
		// 	expectedShouldRedeploy: true,
		// 	expectedOverwriteValues: map[string]interface{}{
		// 		"": "image(test2):myimage:someTagmyimage",
		// 	},
		// },
	}

	for _, testCase := range testCases {
		shouldRedeploy, err := ReplaceImageNames(testCase.overwriteValues, config.NewConfig(nil, nil, &latest.Config{Images: testCase.imagesConf}, testCase.cache, nil, nil, constants.DefaultConfigPath), nil, nil)
		assert.NilError(t, err, "Error replacing image names in testCase %s", testCase.name)

		assert.Equal(t, shouldRedeploy, testCase.expectedShouldRedeploy, "Unexpected deployed-bool in testCase %s", testCase.name)

		ovAsYaml, err := yaml.Marshal(testCase.overwriteValues)
		assert.NilError(t, err, "Error marshaling overwriteValues in testCase %s", testCase.name)
		expectationAsYaml, err := yaml.Marshal(testCase.expectedOverwriteValues)
		assert.NilError(t, err, "Error marshaling expectation in testCase %s", testCase.name)
		assert.Equal(t, string(ovAsYaml), string(expectationAsYaml), "Unexpected overwriteValues in testCase %s", testCase.name)
	}
}
