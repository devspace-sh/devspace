package helm

import (
	"context"
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/remotecache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	fakehelm "github.com/loft-sh/devspace/pkg/devspace/helm/testing"
	helmtypes "github.com/loft-sh/devspace/pkg/devspace/helm/types"
	fakekube "github.com/loft-sh/devspace/pkg/devspace/kubectl/testing"
	"github.com/loft-sh/devspace/pkg/util/log"
	yaml "gopkg.in/yaml.v3"
	"gotest.tools/assert"
	"k8s.io/client-go/kubernetes/fake"
)

type deployTestCase struct {
	name string

	cache       *remotecache.RemoteCache
	forceDeploy bool
	// builtImages    map[string]string
	releasesBefore []*helmtypes.Release
	deployment     string
	releaseName    string
	chart          string
	valuesFiles    []string
	values         map[string]interface{}

	expectedDeployed bool
	expectedErr      string
	expectedCache    *remotecache.RemoteCache
}

func TestDeploy(t *testing.T) {
	testCases := []deployTestCase{
		// TODO: redploy is always true because helmCache.ChartHash != hash is always true
		// {
		// 	name:       "Don't deploy anything",
		// 	deployment: "deploy1",
		// 	cache: &remotecache.RemoteCache{
		// 		Deployments: []remotecache.DeploymentCache{
		// 			{
		// 				Name:                 "deploy1",
		// 				DeploymentConfigHash: "42d471330d96e55ab8d144d52f11e3c319ae2661e50266fa40592bb721689a3a",
		// 				Helm: &remotecache.HelmCache{
		// 					ValuesHash: "ca3d163bab055381827226140568f3bef7eaac187cebd76878e0b63e9e442356",
		// 				},
		// 			},
		// 		},
		// 	},
		// 	releasesBefore: []*helmtypes.Release{
		// 		{
		// 			Name: "deploy1",
		// 		},
		// 	},
		// },
		{
			name:       "Deploy one deployment",
			deployment: "deploy2",
			chart:      ".",
			values: map[string]interface{}{
				"val": "fromVal",
			},
			expectedDeployed: true,
			expectedCache: &remotecache.RemoteCache{
				Deployments: []remotecache.DeploymentCache{
					{
						Name:                 "deploy2",
						DeploymentConfigHash: "038007a9e5018dcf8ffbabad39c1fdc08251e583d7b19e74b765b815d5302444",
						Helm: &remotecache.HelmCache{
							Release:          "deploy2",
							ReleaseNamespace: "testNamespace",
							ReleaseRevision:  "1",
							ValuesHash:       "efd6e101b768968a49f8dba46ef07785ac530ea9f75c4f9ca5733e223b6a4da1",
						},
					},
				},
			},
		},
		{
			name:       "Deploy one deployment with different release name",
			deployment: "deploy2",
			chart:      ".",
			values: map[string]interface{}{
				"val": "fromVal",
			},
			releaseName:      "deploy2-special",
			expectedDeployed: true,
			expectedCache: &remotecache.RemoteCache{
				Deployments: []remotecache.DeploymentCache{
					{
						Name:                 "deploy2-special",
						DeploymentConfigHash: "aca018bdc51747a41d18361a2a678bb64fd36f834d62303b538dcdbb50c5b410",
						Helm: &remotecache.HelmCache{
							Release:          "deploy2-special",
							ReleaseNamespace: "testNamespace",
							ReleaseRevision:  "1",
							ValuesHash:       "efd6e101b768968a49f8dba46ef07785ac530ea9f75c4f9ca5733e223b6a4da1",
						},
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

		if testCase.cache == nil {
			testCase.cache = remotecache.NewCache("testConfig", "testSecret")
		}

		cache := localcache.New(constants.DefaultCacheFolder)
		deployer := &DeployConfig{
			// Kube: kubeClient,
			Helm: &fakehelm.Client{
				Releases: testCase.releasesBefore,
			},
			DeploymentConfig: &latest.DeploymentConfig{
				Name: testCase.deployment,
				Helm: &latest.HelmConfig{
					Chart: &latest.ChartConfig{
						Name: testCase.chart,
					},
					ReleaseName: testCase.releaseName,
					ValuesFiles: testCase.valuesFiles,
					Values:      testCase.values,
				},
			},
		}

		if testCase.expectedCache == nil {
			testCase.expectedCache = testCase.cache
		}
		conf := config.NewConfig(map[string]interface{}{},
			map[string]interface{}{},
			&latest.Config{},
			cache,
			testCase.cache,
			map[string]interface{}{},
			constants.DefaultConfigPath)

		devCtx := devspacecontext.NewContext(context.Background(), nil, log.Discard).WithKubeClient(kubeClient).WithConfig(conf)
		deployed, err := deployer.Deploy(devCtx, testCase.forceDeploy)
		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		for _, deployment := range testCase.cache.Deployments {
			deployment.Helm.OverridesHash = ""
			deployment.Helm.ChartHash = ""
		}
		cacheAsYaml, err := yaml.Marshal(testCase.cache)
		assert.NilError(t, err, "Error marshaling cache in testCase %s", testCase.name)
		expectationAsYaml, err := yaml.Marshal(testCase.expectedCache)

		assert.NilError(t, err, "Error marshaling expected cache in testCase %s", testCase.name)
		assert.Equal(t, string(cacheAsYaml), string(expectationAsYaml), "Unexpected cache in testCase %s", testCase.name)
		assert.Equal(t, deployed, testCase.expectedDeployed, "Unexpected deployed-bool in testCase %s", testCase.name)
	}
}
