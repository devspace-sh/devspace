package helm

import (
	"testing"
	"os"
	"io/ioutil"
	"strings"

	"github.com/otiai10/copy"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	otherhelmpackage "github.com/devspace-cloud/devspace/pkg/devspace/helm"
	
	"k8s.io/client-go/kubernetes/fake"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"gotest.tools/assert"
)

// Test namespace to create
const testNamespace = "test-helm-deploy"

func TestHelmDeployment(t *testing.T) {
	namespace := "testnamespace"
	chartName := "chart"
	valuesFiles := make([]*string, 1)
	valuesFiles0 := "chart"
	valuesFiles[0] = &valuesFiles0

	// 1. Create fake config & generated config
	deployConfig := &latest.DeploymentConfig{
		Name: ptr.String("test-deployment"),
		Helm: &latest.HelmConfig{
			TillerNamespace: &namespace,
			Chart: &latest.ChartConfig{
				Name: &chartName,
			},
			ValuesFiles: &valuesFiles,
		},
	}

	// Create fake devspace config
	testConfig := &latest.Config{
		Deployments: &[]*latest.DeploymentConfig{
			deployConfig,
		},
		// The images config will tell the deployment method to override the image name used in the component above with the tag defined in the generated config below
		Images: &map[string]*latest.ImageConfig{
			"default": &latest.ImageConfig{
				Image: ptr.String("nginx"),
			},
		},
	}
	configutil.SetFakeConfig(testConfig)

	// Create fake generated config
	generatedConfig := &generated.Config{
		ActiveConfig: "default",
		Configs: map[string]*generated.CacheConfig{
			"default": &generated.CacheConfig{
				Images: map[string]*generated.ImageCache{
					"default": &generated.ImageCache{
						Tag: "1.15", // This will be appended to nginx during deploy
					},
				},
			},
		},
	}
	generated.InitDevSpaceConfig(generatedConfig, "default")

	// 2. Write test chart into a temp folder
	dir, err := ioutil.TempDir("", "testDeploy")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}

	copy.Copy("./../../../../examples/minikube", dir)

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}

	// 8. Delete temp folder
	defer os.Chdir(wdBackup)
	defer os.RemoveAll(dir)

	// 3. Init kubectl & create test namespace
	kubeClient := fake.NewSimpleClientset()
	_, err = kubeClient.CoreV1().Namespaces().Create(&k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
	if err != nil {
		t.Fatalf("Error creating namespace: %v", err)
	}

	// 4. Deploy test chart
	helm, err := New(testConfig, kubeClient, deployConfig, log.GetInstance())
	if err != nil {
		t.Fatalf("Error creating helm client: %v", err)
	}
	helm.Helm = otherhelmpackage.NewFakeClient(kubeClient, namespace)
	isDeployed, err := helm.Deploy(generatedConfig.Configs["default"], true, nil)
	if err != nil {
		t.Fatalf("Error deploying chart: %v", err)
	}

	// 5. Validate deployed chart & test .Status function
	assert.Equal(t, true, isDeployed)

	status, err := helm.Status()
	if err != nil {
		t.Fatalf("Error checking status: %v", err)
	}
	if strings.HasPrefix(status.Status, "Deployed") == false {
		t.Fatalf("Unexpected deployment status: %s != Deployed", status.Status)
	}

	// 6. Delete test chart
	err = helm.Delete(generatedConfig.Configs["default"])
	if err != nil {
		t.Fatalf("Error deleting chart: %v", err)
	}

	// 7. Delete test namespace
	err =  kubeClient.CoreV1().Namespaces().Delete(namespace, nil)
	if err != nil {
		t.Fatalf("Error deleting namespace: %v", err)
	}
}
