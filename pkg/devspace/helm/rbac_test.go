package helm

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	"k8s.io/client-go/kubernetes/fake"
)

func createFakeConfig() *latest.Config {
	// Create fake devspace config
	testConfig := &latest.Config{
		Deployments: &[]*latest.DeploymentConfig{
			&latest.DeploymentConfig{
				Name:      ptr.String("test-deployment"),
				Namespace: ptr.String(configutil.TestNamespace),
				Helm: &latest.HelmConfig{
					Chart: &latest.ChartConfig{
						Name: ptr.String("stable/nginx"),
					},
				},
			},
			&latest.DeploymentConfig{
				Name:      ptr.String("test-deployment"),
				Namespace: ptr.String(""),
				Helm: &latest.HelmConfig{
					Chart: &latest.ChartConfig{
						Name: ptr.String("stable/nginx"),
					},
				},
			},
		},
	}
	configutil.SetFakeConfig(testConfig)

	return testConfig
}
func TestCreateTiller(t *testing.T) {
	config := createFakeConfig()

	// Create the fake client.
	client := fake.NewSimpleClientset()

	err := createTillerRBAC(config, client, "tiller-namespace")
	if err != nil {
		t.Fatal(err)
	}
}
