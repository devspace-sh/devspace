package kubectl

import (
	"testing"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	fakekubeloader "github.com/devspace-cloud/devspace/pkg/util/kubeconfig/testing"
	log "github.com/devspace-cloud/devspace/pkg/util/log/testing"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"

	v1beta1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type newClientFromContextTestCase struct {
	name string

	context       string
	namespace     string
	switchContext bool
	contexts      map[string]*clientcmdapi.Context

	expectedErr    bool
	expectedClient *client
}
 
func TestNewClientFromContext(t *testing.T) {
	testCases := []newClientFromContextTestCase{
		{
			name:        "context not there",
			context:     "notThere",
			expectedErr: true,
		},
	}

	for _, testCase := range testCases {
		kubeLoader := &fakekubeloader.Loader{
			RawConfig: &clientcmdapi.Config{
				Contexts: testCase.contexts,
			},
		}
		client, err := NewClientFromContext(testCase.context, testCase.namespace, testCase.switchContext, kubeLoader)

		if !testCase.expectedErr {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected no error in testCase %s", testCase.name)
		}

		clientAsYaml, err := yaml.Marshal(client)
		assert.NilError(t, err, "Error parsing client to yaml in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedClient)
		assert.NilError(t, err, "Error parsing expection to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(clientAsYaml), string(expectedAsYaml), "Unexpected client in testCase %s", testCase.name)
	}
}

type printWarningTestCase struct {
	name string

	generatedConfig *generated.Config
	noWarning       bool
	shouldWait      bool
	clientNamespace string
	clientContext   string

	expectedErr bool
}

func TestPrintWarning(t *testing.T) {
	testCases := []printWarningTestCase{
		{
			name: "Last context is different than current",
			generatedConfig: &generated.Config{
				ActiveProfile: "active",
				Profiles: map[string]*generated.CacheConfig{
					"active": &generated.CacheConfig{
						LastContext: &generated.LastContextConfig{
							Context: "someContext",
						},
					},
				},
			},
			shouldWait:      true,
			clientNamespace: metav1.NamespaceDefault,
		},
		{
			name: "Last namespace is different than current",
			generatedConfig: &generated.Config{
				ActiveProfile: "active",
				Profiles: map[string]*generated.CacheConfig{
					"active": &generated.CacheConfig{
						LastContext: &generated.LastContextConfig{
							Namespace: "someNs",
						},
					},
				},
			},
		},
	}

	second = 0
	defer func() { second = time.Second }()

	for _, testCase := range testCases {
		client := &client{
			namespace:      testCase.clientNamespace,
			currentContext: testCase.clientContext,
		}
		err := client.PrintWarning(testCase.generatedConfig, testCase.noWarning, testCase.shouldWait, &log.FakeLogger{Level: logrus.InfoLevel})

		if !testCase.expectedErr {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected no error in testCase %s", testCase.name)
		}
	}
}


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
