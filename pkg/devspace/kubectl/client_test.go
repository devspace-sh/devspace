package kubectl

import (
	"testing"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/pkg/errors"

	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func createTestConfig() *latest.Config {
	// Create fake devspace config
	testConfig := &latest.Config{}
	configutil.SetFakeConfig(testConfig)

	return testConfig
}

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
	_, err := client.ExtensionsV1beta1().Deployments(configutil.TestNamespace).Create(deploy)
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
	_, err = client.Core().Pods(configutil.TestNamespace).Create(p)
	if err != nil {
		return errors.Wrap(err, "create pod")
	}

	return nil
}

func TestGetPodStatus(t *testing.T) {
	// Create the fake client.
	client := fake.NewSimpleClientset()

	// Inject an event into the fake client.
	err := createTestResources(client)
	if err != nil {
		t.Fatal(err)
	}

	podList, err := client.Core().Pods(configutil.TestNamespace).List(metav1.ListOptions{})
	if err != nil {
		t.Fatalf("error retrieving list: %v", err)
	}

	status := GetPodStatus(&podList.Items[0])
	if status != "Running" {
		t.Fatalf("Unexpected status: %s", status)
	}
}

func TestGetNewestRunningPod(t *testing.T) {
	config := createTestConfig()

	// Create the fake client.
	client := fake.NewSimpleClientset()
	err := createTestResources(client)
	if err != nil {
		t.Fatal(err)
	}

	pod, err := GetNewestRunningPod(config, client, "app.kubernetes.io/name=devspace-app", configutil.TestNamespace, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if pod == nil {
		t.Fatal("Returned pod is nil")
	}
	if pod.Name != "test-pod" {
		t.Fatalf("Returned pod is wrong: %#v", *pod)
	}
}

func TestLogs(t *testing.T) {
	// Create the fake client.
	client := fake.NewSimpleClientset()
	err := createTestResources(client)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Logs(client, configutil.TestNamespace, "test-pod", "test", false, ptr.Int64(100))
	if err != nil && err.Error() != "Request url is empty" {
		t.Fatal(err)
	}
}
