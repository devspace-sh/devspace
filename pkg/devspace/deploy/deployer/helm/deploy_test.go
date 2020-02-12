package helm

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	fakehelm "github.com/devspace-cloud/devspace/pkg/devspace/helm/testing"
	helmtypes "github.com/devspace-cloud/devspace/pkg/devspace/helm/types"
	fakekube "github.com/devspace-cloud/devspace/pkg/devspace/kubectl/testing"
	log "github.com/devspace-cloud/devspace/pkg/util/log/testing"
	yaml "gopkg.in/yaml.v2"
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
	expectedCache    *generated.CacheConfig
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
			name:        "Deploy one deployment",
			deployment:  "deploy2",
			chart:       ".",
			valuesFiles: []string{"."},
			values: map[interface{}]interface{}{
				"val": "fromVal",
			},
			expectedDeployed: true,
			expectedCache: &generated.CacheConfig{
				Deployments: map[string]*generated.DeploymentCache{
					"deploy2": &generated.DeploymentCache{
						DeploymentConfigHash: "2f0fdaa77956604c97de5cb343051fab738ac36052956ae3cb16e8ec529ab154",
					},
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

		if testCase.expectedCache == nil {
			testCase.expectedCache = testCase.cache
		}

		deployed, err := deployer.Deploy(testCase.cache, testCase.forceDeploy, testCase.builtImages)

		assert.Equal(t, deployed, testCase.expectedDeployed, "Unexpected deployed-bool in testCase %s", testCase.name)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		for _, deployment := range testCase.cache.Deployments {
			deployment.HelmOverridesHash = ""
			deployment.HelmChartHash = ""
		}
		cacheAsYaml, err := yaml.Marshal(testCase.cache)
		assert.NilError(t, err, "Error marshaling cache in testCase %s", testCase.name)
		expectationAsYaml, err := yaml.Marshal(testCase.expectedCache)
		assert.NilError(t, err, "Error marshaling expected cache in testCase %s", testCase.name)
		assert.Equal(t, string(cacheAsYaml), string(expectationAsYaml), "Unexpected cache in testCase %s", testCase.name)
	}
}

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
			cache: &generated.CacheConfig{
				Images: map[string]*generated.ImageCache{
					"": &generated.ImageCache{
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
		shouldRedeploy := replaceContainerNames(testCase.overwriteValues, testCase.cache, testCase.imagesConf, testCase.builtImages)

		assert.Equal(t, shouldRedeploy, testCase.expectedShouldRedeploy, "Unexpected deployed-bool in testCase %s", testCase.name)

		ovAsYaml, err := yaml.Marshal(testCase.overwriteValues)
		assert.NilError(t, err, "Error marshaling overwriteValues in testCase %s", testCase.name)
		expectationAsYaml, err := yaml.Marshal(testCase.expectedOverwriteValues)
		assert.NilError(t, err, "Error marshaling expectation in testCase %s", testCase.name)
		assert.Equal(t, string(ovAsYaml), string(expectationAsYaml), "Unexpected overwriteValues in testCase %s", testCase.name)
	}
}
