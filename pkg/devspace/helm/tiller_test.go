package helm

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
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
	_, err := client.ExtensionsV1beta1().Deployments(configutil.TestNamespace).Create(deploy)
	if err != nil {
		return errors.Wrap(err, "create deployment")
	}

	return nil
}

func TestTillerEnsure(t *testing.T) {
	// Create the fake client.
	client := fake.NewSimpleClientset()

	// Inject an event into the fake client.
	err := createTestResources(client)
	if err != nil {
		t.Fatal(err)
	}

	err = ensureTiller(client, configutil.TestNamespace, false)
	if err != nil {
		t.Fatal(err)
	}

	isTillerDeployed := IsTillerDeployed(client, configutil.TestNamespace)
	if isTillerDeployed == false {
		t.Fatal("Expected that tiller is deployed")
	}
}

func TestTillerCreate(t *testing.T) {
	// Create the fake client.
	client := fake.NewSimpleClientset()

	tillerOptions := getTillerOptions(configutil.TestNamespace)

	err := createTiller(client, configutil.TestNamespace, tillerOptions)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTillerDelete(t *testing.T) {
	createFakeConfig()

	// Create the fake client.
	client := fake.NewSimpleClientset()

	// Inject an event into the fake client.
	err := DeleteTiller(client, configutil.TestNamespace)
	if err != nil {
		t.Fatal(err)
	}
}
