package helm

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"

	"k8s.io/client-go/kubernetes/fake"
)

func createFakeConfig() *latest.Config {
	// Create fake devspace config
	testConfig := &latest.Config{
		Deployments: []*latest.DeploymentConfig{
			&latest.DeploymentConfig{
				Name:      "test-deployment",
				Namespace: configutil.TestNamespace,
				Helm: &latest.HelmConfig{
					Chart: &latest.ChartConfig{
						Name: "stable/nginx",
					},
				},
			},
			&latest.DeploymentConfig{
				Name:      "test-deployment",
				Namespace: "",
				Helm: &latest.HelmConfig{
					Chart: &latest.ChartConfig{
						Name: "stable/nginx",
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
	client := &kubectl.Client{
		Client: fake.NewSimpleClientset(),
	}

	err := createTillerRBAC(config, client, "tiller-namespace")
	if err != nil {
		t.Fatal(err)
	}
}
