package helm

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	fakehelm "github.com/devspace-cloud/devspace/pkg/devspace/helm/testing"
	helmtypes "github.com/devspace-cloud/devspace/pkg/devspace/helm/types"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	fakekube "github.com/devspace-cloud/devspace/pkg/devspace/kubectl/testing"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
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
		newTestCase{
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
							Name:    "component-chart",
							Version: "v0.0.8",
							RepoURL: "https://charts.devspace.cloud",
						},
						TillerNamespace: "overwriteTillerNamespace",
						ComponentChart:  ptr.Bool(true),
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		deployer, err := New(testCase.config, testCase.helmClient, testCase.kubeClient, testCase.deployConfig, nil)

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
		deleteTestCase{
			name:   "try to delete without tiller deployed",
			helmV2: true,
		},
		deleteTestCase{
			name: "delete deployment",
			releasesBefore: []*helmtypes.Release{
				&helmtypes.Release{
					Name: "deleteThisRelease",
				},
			},
			deployment: "deleteThisRelease",
			chart:      "deleteThisDeployment",
			cache: &generated.CacheConfig{
				Deployments: map[string]*generated.DeploymentCache{
					"deleteThisDeployment": &generated.DeploymentCache{},
				},
			},
		},
	}

	for _, testCase := range testCases {
		kube := fake.NewSimpleClientset()
		kubeClient := &fakekube.Client{
			Client: kube,
		}

		deployer := &DeployConfig{
			Kube: kubeClient,
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

		if testCase.cache == nil {
			testCase.cache = &generated.CacheConfig{}
		}

		err := deployer.Delete(testCase.cache)

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
