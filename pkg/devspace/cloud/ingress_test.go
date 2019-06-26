package cloud

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"

	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"gotest.tools/assert"
)

func TestCreateIngress(t *testing.T) {
	provider := Provider{}
	kubeClient := fake.NewSimpleClientset()
	testConfig := &latest.Config{
		Cluster: &latest.Cluster{
			Namespace: ptr.String("testNS"),
		},
	}
	kubeClient.CoreV1().Services("testNS").Create(&v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "tiller-deploy",
		},
	})
	kubeClient.CoreV1().Services("testNS").Create(&v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "NoClusterIP",
		},
		Spec: v1.ServiceSpec{
			Type:      v1.ServiceTypeClusterIP,
			ClusterIP: "None",
		},
	})

	err := provider.CreateIngress(testConfig, kubeClient, nil, "")
	assert.Error(t, err, "Couldn't find any active services an ingress could connect to. Please make sure you have a service for your application", "Wrong or no error when creating ingress without any services")

	kubeClient.CoreV1().Services("testNS").Create(&v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "NameForList",
		},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceTypeClusterIP,
			Ports: []v1.ServicePort{
				v1.ServicePort{
					Port: 1,
				},
			},
		},
	})

	err = provider.CreateIngress(testConfig, kubeClient, &Space{Cluster: &Cluster{}}, "")
	assert.Error(t, err, "graphql create ingress path: get token: Provider has no key specified", "Wrong or no error when creating ingress without token")
}
