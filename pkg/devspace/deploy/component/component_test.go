package component

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// Test namespace to create
const testNamespace = "test-component-deploy"

func TestComponentDeployment(t *testing.T) {
	// Create fake devspace config
	testConfig := &latest.Config{
		Deployments: &[]*latest.DeploymentConfig{
			&latest.DeploymentConfig{
				Name: ptr.String("test-deployment"),
				Component: &latest.ComponentConfig{
					Containers: &[]*latest.ContainerConfig{
						{
							Image: ptr.String("nginx"),
						},
					},
					Service: &latest.ServiceConfig{
						Ports: &[]*latest.ServicePortConfig{
							{
								Port: ptr.Int(3000),
							},
						},
					},
				},
			},
		},
		// The images config will tell the deployment method to override the image name used in the component above with the tag defined in the generated config below
		Images: &map[string]*latest.ImageConfig{
			"default": &latest.ImageConfig{
				Image: ptr.String("nginx"),
			},
		},
		Cluster: &latest.Cluster{
			Namespace: ptr.String(testNamespace),
		},
	}
	configutil.SetFakeConfig(testConfig)

	// Create fake generated config
	generatedConfig := &generated.Config{
		ActiveConfig: "default",
		Configs: map[string]*generated.DevSpaceConfig{
			"default": &generated.DevSpaceConfig{
				Deploy: generated.CacheConfig{
					ImageTags: map[string]string{
						"default": "1.15", // This will be appended to nginx during deploy
					},
				},
			},
		},
	}
	generated.InitDevSpaceConfig(generatedConfig, "default")

	// Create the fake client.
	client := fake.NewSimpleClientset()

	// Create test namespace
	_, err := client.Core().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespace,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Init handler
	deployHandler, err := New(client, (*testConfig.Deployments)[0], log.Discard)
	if err != nil {
		t.Fatal(err)
	}

	// Deploy
	err = deployHandler.Deploy(generatedConfig, false, true)
	if err != nil {
		t.Fatal(err)
	}

	// Check if deployment test-deployment is there and a service with the same name
	_, err = client.Core().Services(testNamespace).Get("test-deployment", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ExtensionsV1beta1().Deployments(testNamespace).Get("test-deployment", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// @Florian test deployHandler.Status & deployHandler.Delete
}
