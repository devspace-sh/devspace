package kubectl

import (
	"context"
	"testing"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"
	configTesting "github.com/loft-sh/devspace/pkg/util/kubeconfig/testing"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"github.com/pkg/errors"
	"gotest.tools/assert"

	fakelogger "github.com/loft-sh/devspace/pkg/util/log/testing"
	v1beta1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/clientcmd/api"
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
	_, err := client.AppsV1().Deployments(testNamespace).Create(context.TODO(), deploy, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "create deployment")
	}

	p := &v1.Pod{
		ObjectMeta: podMetadata,
		Spec:       podSpec,
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
			ContainerStatuses: []v1.ContainerStatus{
				{
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
	_, err = client.CoreV1().Pods(testNamespace).Create(context.TODO(), p, metav1.CreateOptions{})
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

	podList, err := kubeClient.CoreV1().Pods(testNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("error retrieving list: %v", err)
	}

	status := GetPodStatus(&podList.Items[0])
	if status != "Running" {
		t.Fatalf("Unexpected status: %s", status)
	}
}

func TestUserAgent(t *testing.T) {
	clusters := make(map[string]*api.Cluster)
	clusters["fake-cluster"] = &api.Cluster{
		Server: "fake-server",
	}

	contexts := make(map[string]*api.Context)
	contexts["fake-context"] = &api.Context{
		Cluster: "fake-cluster",
	}

	loader := &configTesting.Loader{
		RawConfig: &api.Config{
			Clusters: clusters,
			Contexts: contexts,
		},
	}
	client, err := NewClientFromContext("fake-context", "", false, loader)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, client.RestConfig().UserAgent, "DevSpace Version "+upgrade.GetVersion())
}

type testCaseContext struct {
	selectContext string
	namespace     string
}

func TestCheckKubeContext(t *testing.T) {
	context1 := "context1"
	context2 := "context2"
	ns1 := "n1"
	ns2 := "ns2"

	localCache := &localcache.LocalCache{
		LastContext: &localcache.LastContextConfig{
			Context:   context1,
			Namespace: ns1,
		},
	}

	testCases := []testCaseContext{
		{
			// selecting last context
			selectContext: context1,
			namespace:     ns1, // ns2 should be reverted to back to ns1
		},
		{
			// selecting current context
			selectContext: context2,
			namespace:     ns2, // same ns should be used
		},
	}

	clusters := make(map[string]*api.Cluster)
	clusters["cluster1"] = &api.Cluster{
		Server: "server1",
	}
	clusters["cluster2"] = &api.Cluster{
		Server: "server2",
	}

	contexts := make(map[string]*api.Context)
	contexts[context1] = &api.Context{
		Cluster: "cluster1",
	}
	contexts[context2] = &api.Context{
		Cluster: "cluster2",
	}

	loader := &configTesting.Loader{
		RawConfig: &api.Config{
			Clusters: clusters,
			Contexts: contexts,
		},
	}

	fakeLogger := fakelogger.NewFakeLogger()
	fakeLogger.SetLevel(4)

	for _, tc := range testCases {
		// creating client
		client, err := NewClientFromContext(context2, ns2, false, loader)
		if err != nil {
			t.Fatal(err)
		}
		assert.Assert(t, client.CurrentContext() == context2)
		assert.Assert(t, client.Namespace() == ns2)

		fakeLogger.SetAnswer(tc.selectContext)

		// checking kubeContext and reseting the client
		isTerminalIn = true
		client, err = CheckKubeContext(client, localCache, false, false, true, fakeLogger)
		if err != nil {
			t.Fatal(err)
		}
		assert.Assert(t, client.CurrentContext() == tc.selectContext)
		assert.Assert(t, client.Namespace() == tc.namespace)
	}
}

func TestCheckKubeContextNamespace(t *testing.T) {
	context1 := "context1"
	ns1 := "n1"
	ns2 := "ns2"

	localCache := &localcache.LocalCache{
		LastContext: &localcache.LastContextConfig{
			Context:   context1,
			Namespace: ns1,
		},
	}
	clusters := make(map[string]*api.Cluster)
	clusters["cluster1"] = &api.Cluster{
		Server: "server1",
	}

	contexts := make(map[string]*api.Context)
	contexts[context1] = &api.Context{
		Cluster: "cluster1",
	}

	loader := &configTesting.Loader{
		RawConfig: &api.Config{
			Clusters: clusters,
			Contexts: contexts,
		},
	}

	fakeLogger := fakelogger.NewFakeLogger()
	fakeLogger.SetLevel(4)

	ns := []string{ns1, ns2}
	for _, n := range ns {
		// creating client
		client, err := NewClientFromContext(context1, ns2, false, loader)
		if err != nil {
			t.Fatal(err)
		}
		assert.Assert(t, client.CurrentContext() == context1)
		assert.Assert(t, client.Namespace() == ns2)

		fakeLogger.SetAnswer(n)
		// checking kubeContext and reseting the client
		isTerminalIn = true
		client, err = CheckKubeContext(client, localCache, false, false, true, fakeLogger)
		if err != nil {
			t.Fatal(err)
		}
		assert.Assert(t, client.CurrentContext() == context1)
		assert.Assert(t, client.Namespace() == n)
	}
}
