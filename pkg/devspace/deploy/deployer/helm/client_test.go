package helm

import (
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	fakehelm "github.com/loft-sh/devspace/pkg/devspace/helm/testing"
	helmtypes "github.com/loft-sh/devspace/pkg/devspace/helm/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	fakekube "github.com/loft-sh/devspace/pkg/devspace/kubectl/testing"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	yaml "gopkg.in/yaml.v2"
	"gotest.tools/assert"
	"k8s.io/client-go/kubernetes/fake"
)

type newTestCase struct {
	name string

	config       *latest.Config
	helmClient   helmtypes.Client
	kubeClient   kubectl.Client
	deployConfig *latest.DeploymentConfig

	expectedDeployer DeployConfig
	expectedErr      string
}

func TestNew(t *testing.T) {
	testCases := []newTestCase{
		{
			name:       "new client with kubeClient and chart",
			kubeClient: &fakekube.Client{},
			deployConfig: &latest.DeploymentConfig{
				Helm: &latest.HelmConfig{
					TillerNamespace: "overwriteTillerNamespace",
					ComponentChart:  ptr.Bool(true),
				},
			},
			expectedDeployer: DeployConfig{
				Kube:            &fakekube.Client{},
				TillerNamespace: "overwriteTillerNamespace",
				DeploymentConfig: &latest.DeploymentConfig{
					Helm: &latest.HelmConfig{
						Chart: &latest.ChartConfig{
							Name:    DevSpaceChartConfig.Name,
							Version: DevSpaceChartConfig.Version,
							RepoURL: DevSpaceChartConfig.RepoURL,
						},
						TillerNamespace: "overwriteTillerNamespace",
						ComponentChart:  ptr.Bool(true),
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		deployer, err := New(config.NewConfig(nil, testCase.config, nil, nil, constants.DefaultConfigPath), nil, testCase.helmClient, testCase.kubeClient, testCase.deployConfig, nil)
		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		statusAsYaml, err := yaml.Marshal(deployer)
		assert.NilError(t, err, "Error marshaling deployer in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedDeployer)
		assert.NilError(t, err, "Error marshaling expected deployer in testCase %s", testCase.name)
		assert.Equal(t, string(statusAsYaml), string(expectedAsYaml), "Unexpected deployer in testCase %s", testCase.name)
	}
}

type deleteTestCase struct {
	name string

	helmV2         bool
	cache          *generated.CacheConfig
	releasesBefore []*helmtypes.Release
	deployment     string
	chart          string

	expectedDeployments map[string]*generated.DeploymentCache
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
			cache: &generated.CacheConfig{
				Deployments: map[string]*generated.DeploymentCache{
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
			testCase.cache = &generated.CacheConfig{}
		}
		cache := generated.New()
		cache.Profiles[""] = testCase.cache
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
					V2: testCase.helmV2,
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
