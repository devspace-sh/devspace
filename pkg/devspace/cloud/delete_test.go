package cloud

import (
	"regexp"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	testconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/testing"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	fakekubeconfig "github.com/devspace-cloud/devspace/pkg/util/kubeconfig/testing"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
	"k8s.io/client-go/tools/clientcmd/api"
)

type deleteKubeContextTestCase struct {
	name string

	space        *latest.Space
	spacesBefore map[int]*latest.SpaceCache

	expectedErr    string
	expectedConfig api.Config
	expectedSpaces map[int]*latest.SpaceCache
}

func TestDeleteKubeContext(t *testing.T) {
	testCases := []deleteKubeContextTestCase{
		deleteKubeContextTestCase{
			name: "Delete",
			space: &latest.Space{
				SpaceID:      2,
				Name:         "deletedSpace",
				ProviderName: config.DevSpaceCloudProviderName,
			},
			spacesBefore: map[int]*latest.SpaceCache{
				2: &latest.SpaceCache{},
			},
		},
	}

	for _, testCase := range testCases {
		rawConfig := &api.Config{
			AuthInfos: map[string]*api.AuthInfo{},
			Clusters:  map[string]*api.Cluster{},
			Contexts: map[string]*api.Context{
				"devspace-deletedspace": &api.Context{},
			},
		}

		provider := &provider{
			Provider: latest.Provider{
				Name:   "providerName",
				Spaces: testCase.spacesBefore,
			},
			kubeLoader: &fakekubeconfig.Loader{
				RawConfig: rawConfig,
			},
			loader: testconfig.NewLoader(&latest.Config{}),
		}

		err := provider.DeleteKubeContext(testCase.space)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		configAsYaml, err := yaml.Marshal(rawConfig)
		assert.NilError(t, err, "Error parsing config to yaml in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedConfig)
		assert.NilError(t, err, "Error parsing config expection to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(configAsYaml), string(expectedAsYaml), "Unexpected spaces in testCase %s", testCase.name)

		spacesAsYaml, err := yaml.Marshal(provider.Provider.Spaces)
		assert.NilError(t, err, "Error parsing spaces to yaml in testCase %s", testCase.name)
		expectedAsYaml, err = yaml.Marshal(testCase.expectedSpaces)
		assert.NilError(t, err, "Error parsing spaces expection to yaml in testCase %s", testCase.name)
		lineWithTimestamp := regexp.MustCompile("(?m)[\r\n]+^.*expires.*$")
		spacesString := lineWithTimestamp.ReplaceAllString(string(spacesAsYaml), "")
		expectedString := lineWithTimestamp.ReplaceAllString(string(expectedAsYaml), "")
		assert.Equal(t, spacesString, expectedString, "Unexpected spaces in testCase %s", testCase.name)
	}
}
