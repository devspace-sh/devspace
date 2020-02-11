package kubectl

import (
	"testing"
	"time"

	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/pkg/errors"

	v1beta1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

const testNamespace = "test"

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
		ObjectMeta: metav1.ObjectMeta{Name: "test-deployment"},
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
	_, err := client.AppsV1().Deployments(testNamespace).Create(deploy)
	if err != nil {
		return errors.Wrap(err, "create deployment")
	}

	p := &v1.Pod{
		ObjectMeta: podMetadata,
		Spec:       podSpec,
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
			ContainerStatuses: []v1.ContainerStatus{
				v1.ContainerStatus{
					Name:  "test",
					Ready: true,
					Image: "nginx",
					State: v1.ContainerState{
						Running: &v1.ContainerStateRunning{
							StartedAt: metav1.NewTime(time.Now()),
						},
					},
				},
			},
		},
	}
	_, err = client.CoreV1().Pods(testNamespace).Create(p)
	if err != nil {
		return errors.Wrap(err, "create pod")
	}

	return nil
}

func TestGetPodStatus(t *testing.T) {
	// Create the fake client.
	kubeClient := fake.NewSimpleClientset()

	// Inject an event into the fake client.
	err := createTestResources(kubeClient)
	if err != nil {
		t.Fatal(err)
	}

	podList, err := kubeClient.CoreV1().Pods(testNamespace).List(metav1.ListOptions{})
	if err != nil {
		t.Fatalf("error retrieving list: %v", err)
	}

	status := GetPodStatus(&podList.Items[0])
	if status != "Running" {
		t.Fatalf("Unexpected status: %s", status)
	}
}
