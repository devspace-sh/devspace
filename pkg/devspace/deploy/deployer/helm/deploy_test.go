package helm

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	fakehelm "github.com/devspace-cloud/devspace/pkg/devspace/helm/testing"
	helmtypes "github.com/devspace-cloud/devspace/pkg/devspace/helm/types"
	fakekube "github.com/devspace-cloud/devspace/pkg/devspace/kubectl/testing"
	log "github.com/devspace-cloud/devspace/pkg/util/log/testing"
	"gotest.tools/assert"
	"k8s.io/client-go/kubernetes/fake"
)

type deployTestCase struct {
	name string

	cache          *generated.CacheConfig
	forceDeploy    bool
	builtImages    map[string]string
	releasesBefore []*helmtypes.Release
	deployment     string
	chart          string
	valuesFiles    []string
	values         map[interface{}]interface{}

	expectedDeployed bool
	expectedErr      string
}

func TestDeploy(t *testing.T) {
	testCases := []deployTestCase{
		deployTestCase{
			name:       "Don't deploy anything",
			deployment: "deploy1",
			cache: &generated.CacheConfig{
				Deployments: map[string]*generated.DeploymentCache{
					"deploy1": &generated.DeploymentCache{
						DeploymentConfigHash: "42d471330d96e55ab8d144d52f11e3c319ae2661e50266fa40592bb721689a3a",
					},
				},
			},
			releasesBefore: []*helmtypes.Release{
				&helmtypes.Release{
					Name: "deploy1",
				},
			},
		},
		deployTestCase{
			name:        "",
			deployment:  "deploy2",
			chart:       "/",
			valuesFiles: []string{"/"},
			values: map[interface{}]interface{}{
				"val": "fromVal",
			},
			expectedDeployed: true,
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
					ValuesFiles: testCase.valuesFiles,
					Values:      testCase.values,
				},
			},
			config: &latest.Config{},
			Log:    &log.FakeLogger{},
		}

		if testCase.cache == nil {
			testCase.cache = &generated.CacheConfig{
				Deployments: map[string]*generated.DeploymentCache{},
			}
		}

		deployed, err := deployer.Deploy(testCase.cache, testCase.forceDeploy, testCase.builtImages)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		assert.Equal(t, deployed, testCase.expectedDeployed, "Unexpected deployed-bool in testCase %s", testCase.name)
	}
}
