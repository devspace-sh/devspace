package util

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type replaceContainerNamesTestCase struct {
	name string

	overwriteValues map[interface{}]interface{}
	cache           *generated.CacheConfig
	imagesConf      map[string]*latest.ImageConfig
	builtImages     map[string]string

	expectedShouldRedeploy  bool
	expectedOverwriteValues map[interface{}]interface{}
}

func TestReplaceContainerNames(t *testing.T) {
	testCases := []replaceContainerNamesTestCase{
		replaceContainerNamesTestCase{
			name: "invalid image name",
			overwriteValues: map[interface{}]interface{}{
				"": "",
			},
			cache: &generated.CacheConfig{
				Images: map[string]*generated.ImageCache{
					"": &generated.ImageCache{},
				},
			},
			expectedOverwriteValues: map[interface{}]interface{}{
				"": "",
			},
		},
		replaceContainerNamesTestCase{
			name: "Image not in cache",
			overwriteValues: map[interface{}]interface{}{
				"": "myimage",
			},
			cache: &generated.CacheConfig{
				Images: map[string]*generated.ImageCache{},
			},
			expectedOverwriteValues: map[interface{}]interface{}{
				"": "myimage",
			},
		},
		replaceContainerNamesTestCase{
			name: "Image in cache",
			overwriteValues: map[interface{}]interface{}{
				"": "myimage",
			},
			imagesConf: map[string]*latest.ImageConfig{
				"test": &latest.ImageConfig{},
			},
			cache: &generated.CacheConfig{
				Images: map[string]*generated.ImageCache{
					"test": &generated.ImageCache{
						ImageName: "myimage",
						Tag:       "someTag",
					},
				},
			},
			builtImages: map[string]string{
				"myimage": "",
			},
			expectedShouldRedeploy: true,
			expectedOverwriteValues: map[interface{}]interface{}{
				"": "myimage:someTag",
			},
		},
	}

	for _, testCase := range testCases {
		shouldRedeploy := ReplaceImageNames(testCase.overwriteValues, testCase.cache, testCase.imagesConf, testCase.builtImages, nil)

		assert.Equal(t, shouldRedeploy, testCase.expectedShouldRedeploy, "Unexpected deployed-bool in testCase %s", testCase.name)

		ovAsYaml, err := yaml.Marshal(testCase.overwriteValues)
		assert.NilError(t, err, "Error marshaling overwriteValues in testCase %s", testCase.name)
		expectationAsYaml, err := yaml.Marshal(testCase.expectedOverwriteValues)
		assert.NilError(t, err, "Error marshaling expectation in testCase %s", testCase.name)
		assert.Equal(t, string(ovAsYaml), string(expectationAsYaml), "Unexpected overwriteValues in testCase %s", testCase.name)
	}
}
