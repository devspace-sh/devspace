package build

import (
	"reflect"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/build/builder/custom"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type createBuilderTestCase struct {
	name string

	imageConfigName string
	imageConf       *latest.ImageConfig
	imageTag        string
	options         Options

	expectedErr     string
	expectedBuilder interface{}
}

func TestCreateBuilder(t *testing.T) {
	testCases := []createBuilderTestCase{
		createBuilderTestCase{
			name:            "Create custom builder",
			imageConfigName: "imageConfigName",
			imageConf: &latest.ImageConfig{
				Build: &latest.BuildConfig{
					Custom: &latest.CustomConfig{},
				},
			},
			imageTag: "imageTag",
			expectedBuilder: custom.NewBuilder("imageConfigName", &latest.ImageConfig{
				Build: &latest.BuildConfig{
					Custom: &latest.CustomConfig{},
				},
			}, "imageTag"),
		},
	}

	fakeBuilder = nil

	for _, testCase := range testCases {
		controller := &controller{}

		builder, err := controller.createBuilder(testCase.imageConfigName, testCase.imageConf, testCase.imageTag, &testCase.options, log.Discard)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error updating all in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error from UpdateALl in testCase %s", testCase.name)
		}

		builderAsYaml, err := yaml.Marshal(builder)
		assert.NilError(t, err, "Error marshaling builder in testCase %s", testCase.name)
		expectationAsYaml, err := yaml.Marshal(testCase.expectedBuilder)
		assert.NilError(t, err, "Error marshaling expected builder in testCase %s", testCase.name)
		assert.Equal(t, string(builderAsYaml), string(expectationAsYaml), "Unexpected cache in testCase %s", testCase.name)
		assert.Equal(t, reflect.TypeOf(builder), reflect.TypeOf(testCase.expectedBuilder), "Unexpected cache type in testCase %s", testCase.name)
	}
}
