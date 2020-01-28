package cloud

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	fakekubeconfig "github.com/devspace-cloud/devspace/pkg/util/kubeconfig/testing"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestGetKubeContextNameFromSpace(t *testing.T) {
	assert.Equal(t, GetKubeContextNameFromSpace("space:Name", "provider.Name"), DevSpaceKubeContextName+"-provider-name-space-name", "Wrong KubeContextName returned")
}

type updateKubeConfigTestCase struct {
	name string

	context        string
	serviceAccount *latest.ServiceAccount
	spaceID        int
	setActive      bool

	expectedErr    string
	expectedConfig api.Config
}

func TestUpdateKubeConfig(t *testing.T) {
	testCases := []updateKubeConfigTestCase{
		updateKubeConfigTestCase{
			name: "Save",
			serviceAccount: &latest.ServiceAccount{
				Token: "saParamCert",
			},
			setActive: true,
			context:   "contextParam",
			spaceID:   2,
			expectedConfig: api.Config{
				Clusters: map[string]*api.Cluster{
					"contextParam": &api.Cluster{},
				},
				AuthInfos: map[string]*api.AuthInfo{
					"contextParam": &api.AuthInfo{
						Exec: &api.ExecConfig{
							APIVersion: "client.authentication.k8s.io/v1alpha1",
							Command:    "devspace",
							Args:       []string{"use", "space", "--provider", "providerName", "--space-id", "2", "--get-token", "--silent"},
						},
					},
				},
				Contexts: map[string]*api.Context{
					"contextParam": &api.Context{
						Cluster:  "contextParam",
						AuthInfo: "contextParam",
					},
				},
				CurrentContext: "contextParam",
			},
		},
	}

	for _, testCase := range testCases {
		rawConfig := &api.Config{
			AuthInfos: map[string]*api.AuthInfo{},
			Clusters:  map[string]*api.Cluster{},
			Contexts:  map[string]*api.Context{},
		}

		provider := &provider{
			Provider: latest.Provider{
				Name: "providerName",
			},
			kubeLoader: &fakekubeconfig.Loader{
				RawConfig: rawConfig,
			},
		}

		err := provider.UpdateKubeConfig(testCase.context, testCase.serviceAccount, testCase.spaceID, testCase.setActive)

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
	}

}
