package helm

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	fakehelm "github.com/loft-sh/devspace/pkg/devspace/helm/testing"
	helmtypes "github.com/loft-sh/devspace/pkg/devspace/helm/types"
	fakekube "github.com/loft-sh/devspace/pkg/devspace/kubectl/testing"
	yaml "gopkg.in/yaml.v3"
	"gotest.tools/assert"
	"k8s.io/client-go/kubernetes/fake"
)

type deleteTestCase struct {
	name string

	helmV2         bool
	cache          *localcache.LocalCache
	releasesBefore []*helmtypes.Release
	deployment     string
	chart          string

	expectedDeployments map[string]*localcache.DeploymentCache
	expectedErr         string
}

func TestDelete(t *testing.T) {
	testCases := []deleteTestCase{
		{
			name:   "try to delete without tiller deployed",
			helmV2: true,
		},
		{
			name: "delete deployment",
			releasesBefore: []*helmtypes.Release{
				{
					Name: "deleteThisRelease",
				},
			},
			deployment: "deleteThisRelease",
			chart:      "deleteThisDeployment",
			cache: &localcache.LocalCache{
				Deployments: map[string]localcache.DeploymentCache{
					"deleteThisDeployment": {},
				},
			},
		},
	}

	for _, testCase := range testCases {
		kube := fake.NewSimpleClientset()
		kubeClient := &fakekube.Client{
			Client: kube,
		}

		if testCase.cache == nil {
			testCase.cache = &localcache.LocalCache{}
		}
		cache := localcache.New()
		deployer := &DeployConfig{
			config: config.NewConfig(nil, latest.NewRaw(), cache, nil, constants.DefaultConfigPath),
			Kube:   kubeClient,
			Helm: &fakehelm.Client{
				Releases: testCase.releasesBefore,
			},
			DeploymentConfig: &latest.DeploymentConfig{
				Name: testCase.deployment,
				Helm: &latest.HelmConfig{
					Chart: &latest.ChartConfig{
						Name: testCase.chart,
					},
				},
			},
		}

		err := deployer.Delete()
		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		statusAsYaml, err := yaml.Marshal(testCase.cache.Deployments)
		assert.NilError(t, err, "Error marshaling status in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedDeployments)
		assert.NilError(t, err, "Error marshaling expected status in testCase %s", testCase.name)
		assert.Equal(t, string(statusAsYaml), string(expectedAsYaml), "Unexpected status in testCase %s", testCase.name)
	}
}
