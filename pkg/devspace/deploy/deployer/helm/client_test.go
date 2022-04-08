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
	"github.com/loft-sh/devspace/pkg/util/log"

	helmtypes "github.com/loft-sh/devspace/pkg/devspace/helm/types"
	fakekube "github.com/loft-sh/devspace/pkg/devspace/kubectl/testing"
	"gotest.tools/assert"
	"k8s.io/client-go/kubernetes/fake"
)

type deleteTestCase struct {
	name string

	cache          *remotecache.RemoteCache
	releasesBefore []*helmtypes.Release
	deployment     string
	chart          string

	// expectedDeployments map[string]*remotecache.DeploymentCache
	expectedErr string
}

func TestDelete(t *testing.T) {
	testCases := []deleteTestCase{
		{
			name: "delete deployment",
			releasesBefore: []*helmtypes.Release{
				{
					Name: "deleteThisRelease",
				},
			},
			deployment: "deleteThisRelease",
			chart:      "deleteThisDeployment",
			cache: &remotecache.RemoteCache{
				Deployments: []remotecache.DeploymentCache{
					{
						Name: "deleteThisDeployment",
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
			testCase.cache = &remotecache.RemoteCache{}
		}

		cg := config.NewConfig(map[string]interface{}{}, map[string]interface{}{}, latest.NewRaw(), localcache.New(""), testCase.cache, map[string]interface{}{}, constants.DefaultConfigsPath)
		devContext := devspacecontext.NewContext(context.Background(), nil, log.Discard).WithConfig(cg).WithKubeClient(kubeClient)

		err := Delete(devContext, testCase.deployment)
		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		// statusAsYaml, err := yaml.Marshal(testCase.cache.Deployments)
		// assert.NilError(t, err, "Error marshaling status in testCase %s", testCase.name)
		// expectedAsYaml, err := yaml.Marshal(testCase.expectedDeployments)
		// assert.NilError(t, err, "Error marshaling expected status in testCase %s", testCase.name)
		// assert.Equal(t, string(statusAsYaml), string(expectedAsYaml), "Unexpected status in testCase %s", testCase.name)
	}
}
