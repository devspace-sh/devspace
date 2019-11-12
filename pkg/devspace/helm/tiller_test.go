package helm

/*import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/pkg/errors"
	v1beta1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"gotest.tools/assert"
)

func createTestResources(client kubernetes.Interface) error {
	podMetadata := metav1.ObjectMeta{
		Name: "test-pod",
		Labels: map[string]string{
			"app.kubernetes.io/name": "devspace-app",
		},
	}
	podSpec := v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:  "test",
				Image: "nginx",
			},
		},
	}

	deploy := &v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: TillerDeploymentName},
		Spec: v1beta1.DeploymentSpec{
			Replicas: ptr.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": "devspace-app",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: podMetadata,
				Spec:       podSpec,
			},
		},
		Status: v1beta1.DeploymentStatus{
			AvailableReplicas:  1,
			ObservedGeneration: 1,
			ReadyReplicas:      1,
			Replicas:           1,
			UpdatedReplicas:    1,
		},
	}
	_, err := client.AppsV1().Deployments(configutil.TestNamespace).Create(deploy)
	if err != nil {
		return errors.Wrap(err, "create deployment")
	}

	return nil
}

func TestTillerEnsure(t *testing.T) {
	t.Skip("Skip this test for now because helm is creating tiller with extensions v1beta1 but we expect apps v1")
	config := createFakeConfig()

	// Create the fake client.
	client := &kubectl.Client{
		Client: fake.NewSimpleClientset(),
	}

	// Inject an event into the fake client.
	err := createTestResources(client.KubeClient())
	if err != nil {
		t.Fatal(err)
	}

	err = ensureTiller(config, client, configutil.TestNamespace, true, log.Discard)
	if err != nil {
		t.Fatal(err)
	}

	isTillerDeployed := IsTillerDeployed(config, client, configutil.TestNamespace)
	if isTillerDeployed == false {
		t.Fatal("Expected that tiller is deployed")
	}

	//Break deployment
	deployment, err := client.KubeClient().AppsV1().Deployments(configutil.TestNamespace).Get(TillerDeploymentName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Error breaking deployment: %v", err)
	}
	deployment.Status.Replicas = 1
	deployment.Status.ReadyReplicas = 2
	client.KubeClient().AppsV1().Deployments(configutil.TestNamespace).Update(deployment)

	isTillerDeployed = IsTillerDeployed(config, client, configutil.TestNamespace)
	assert.Equal(t, false, isTillerDeployed, "Tiller declared deployed despite deployment being broken")
}

func TestTillerCreate(t *testing.T) {
	config := createFakeConfig()

	// Create the fake client.
	client := &kubectl.Client{
		Client: fake.NewSimpleClientset(),
	}

	tillerOptions := getTillerOptions(configutil.TestNamespace)

	err := createTiller(config, client, configutil.TestNamespace, tillerOptions, log.Discard)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTillerDelete(t *testing.T) {
	config := createFakeConfig()

	// Create the fake client.
	client := &kubectl.Client{
		Client: fake.NewSimpleClientset(),
	}

	// Inject an event into the fake client.
	err := DeleteTiller(config, client, configutil.TestNamespace)
	if err != nil {
		t.Fatal(err)
	}
}*/
