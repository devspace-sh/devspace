package util

import (
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
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
		{
			name: "invalid image name",
			overwriteValues: map[interface{}]interface{}{
				"": "",
			},
			cache: &generated.CacheConfig{
				Images: map[string]*generated.ImageCache{
					"": {},
				},
			},
			expectedOverwriteValues: map[interface{}]interface{}{
				"": "",
			},
		},
		{
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
		{
			name: "Image in cache",
			overwriteValues: map[interface{}]interface{}{
				"": "myimage",
			},
			imagesConf: map[string]*latest.ImageConfig{
				"test": {
					Image: "myimage",
				},
			},
			cache: &generated.CacheConfig{
				Images: map[string]*generated.ImageCache{
					"test": {
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
		{
			name: "Replace image & tag helpers",
			overwriteValues: map[interface{}]interface{}{
				"": "image(test):tag(test)",
			},
			imagesConf: map[string]*latest.ImageConfig{
				"test": {
					Image: "myimage",
				},
			},
			cache: &generated.CacheConfig{
				Images: map[string]*generated.ImageCache{
					"test": {
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
		{
			name: "Do not replace unknown tag helpers",
			overwriteValues: map[interface{}]interface{}{
				"": "tag(test2):image(test):tag(test)image(test)",
			},
			imagesConf: map[string]*latest.ImageConfig{
				"test": {
					Image: "myimage",
				},
			},
			cache: &generated.CacheConfig{
				Images: map[string]*generated.ImageCache{
					"test": {
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
				"": "tag(test2):myimage:someTagmyimage",
			},
		},
		{
			name: "Do not replace unknown image helpers",
			overwriteValues: map[interface{}]interface{}{
				"": "image(test2):image(test):tag(test)image(test)",
			},
			imagesConf: map[string]*latest.ImageConfig{
				"test": {
					Image: "myimage",
				},
			},
			cache: &generated.CacheConfig{
				Images: map[string]*generated.ImageCache{
					"test": {
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
				"": "image(test2):myimage:someTagmyimage",
			},
		},
	}

	for _, testCase := range testCases {
		cache := generated.New()
		cache.Profiles[""] = testCase.cache
		shouldRedeploy, err := ReplaceImageNames(testCase.overwriteValues, config.NewConfig(nil, &latest.Config{Images: testCase.imagesConf}, cache, nil, constants.DefaultConfigPath), nil, testCase.builtImages, nil)
		assert.NilError(t, err, "Error replacing image names in testCase %s", testCase.name)

		assert.Equal(t, shouldRedeploy, testCase.expectedShouldRedeploy, "Unexpected deployed-bool in testCase %s", testCase.name)

		ovAsYaml, err := yaml.Marshal(testCase.overwriteValues)
		assert.NilError(t, err, "Error marshaling overwriteValues in testCase %s", testCase.name)
		expectationAsYaml, err := yaml.Marshal(testCase.expectedOverwriteValues)
		assert.NilError(t, err, "Error marshaling expectation in testCase %s", testCase.name)
		assert.Equal(t, string(ovAsYaml), string(expectationAsYaml), "Unexpected overwriteValues in testCase %s", testCase.name)
	}
}
