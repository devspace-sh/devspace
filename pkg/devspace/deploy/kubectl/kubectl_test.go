package kubectl

import (
	"testing"
	"os"
	"io/ioutil"
	//"strings"

	"github.com/otiai10/copy"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	
	"k8s.io/client-go/kubernetes/fake"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"gotest.tools/assert"
)

// Test namespace to create
const testNamespace = "test-kubectl-deploy"

// Test namespace to create
const testKustomizeNamespace = "test-kubectl-kustomize-deploy"

// @MoreTests
//When kubectl is testable, test it

func TestKubectlManifests(t *testing.T) {
	t.Skip("Not yet testable")
	namespace := "testnamespace"
	manifests := make([]*string, 1)
	manifests0 := "kube"
	manifests[0] = &manifests0

	flags := make([]*string, 1)
	flags0 := "--dry-run"
	flags[0] = &flags0
	// 1. Create fake config & generated config

	// Create fake devspace config
	deploymentConfig := &latest.DeploymentConfig{
		Name: ptr.String("test-deployment"),
		Kubectl: &latest.KubectlConfig{
			Manifests: &manifests,
			Flags: &flags,
		},
	}
	testConfig := &latest.Config{
		Deployments: &[]*latest.DeploymentConfig{
			deploymentConfig,
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

	// 2. Write test manifests into a temp folder
	dir, err := ioutil.TempDir("", "testDeploy")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}

	copy.Copy("./../../../../examples/microservices/node", dir)

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

	// 4. Deploy manifests
	deployConfig, err := New(testConfig, kubeClient, deploymentConfig, log.GetInstance())
	if err != nil {
		t.Fatalf("Error creating deployConfig: %v", err)
	}

	isDeployed, err := deployConfig.Deploy(generatedConfig.Configs["default"], true, nil)
	if err != nil {
		t.Fatalf("Error deploying chart: %v", err)
	}
	assert.Equal(t, true, isDeployed, "Manifest is not deployed. No errors returned.")
	// 5. Validate manifests
	// 6. Delete manifests
	// 7. Delete test namespace
}

func TestKubectlManifestsWithKustomize(t *testing.T) {
	// @MoreTests
	// 1. Create fake config & generated config
	// 2. Write test kustomize files (see examples) into a temp folder
	// 3. Init kubectl & create test namespace
	// 4. Deploy files
	// 5. Validate deployed resources
	// 6. Delete deployed files
	// 7. Delete test namespace
	// 8. Delete temp folder
}
