package kubectl

import (
	"net"
	"testing"

	log "github.com/devspace-cloud/devspace/pkg/util/log/testing"
	"gotest.tools/assert"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type isPrivateIPTestCase struct {
	name string

	ip string

	expectedIsPrivate bool
}

func TestIsPrivateIP(t *testing.T) {
	testCases := []isPrivateIPTestCase{
		{
			name:              "Private IP",
			ip:                "127.0.0.0/8",
			expectedIsPrivate: true,
		},
		{
			name:              "Google IP",
			ip:                "64.233.160.0",
			expectedIsPrivate: false,
		},
	}

	for _, testCase := range testCases {
		ip, _, _ := net.ParseCIDR(testCase.ip)
		assert.Equal(t, IsPrivateIP(ip), testCase.expectedIsPrivate, "Unexpected resut in testCase %s", testCase.name)
	}
}

type ensureDefaultNamespaceTestCase struct {
	name string

	namespace  string
	namespaces []string

	expectedErr        bool
	expectedNamespaces []string
}

func TestEnsureDefaultNamespace(t *testing.T) {
	testCases := []ensureDefaultNamespaceTestCase{
		{
			name:               "Namespace is already there",
			namespace:          "exists",
			namespaces:         []string{"exists"},
			expectedNamespaces: []string{"exists"},
		},
		{
			name:               "Namespace is new",
			namespace:          "new",
			namespaces:         []string{"exists"},
			expectedNamespaces: []string{"exists", "new"},
		},
	}

	for _, testCase := range testCases {
		kubeClient := fake.NewSimpleClientset()
		for _, namespace := range testCase.namespaces {
			kubeClient.CoreV1().Namespaces().Create(&k8sv1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			})
		}
		client := &client{
			Client:    kubeClient,
			namespace: testCase.namespace,
		}

		err := client.EnsureDefaultNamespace(&log.FakeLogger{})

		if !testCase.expectedErr {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected no error in testCase %s", testCase.name)
		}

		namespaces, _ := kubeClient.CoreV1().Namespaces().List(metav1.ListOptions{})
		assert.Equal(t, len(namespaces.Items), len(testCase.expectedNamespaces), "Unexpected number of namespaces after call in testCase %s", testCase.name)
		for index, namespace := range testCase.expectedNamespaces {
			assert.Equal(t, namespaces.Items[index].ObjectMeta.Name, namespace, "Unexpected namespace at index %d in testCase %s", index, testCase.name)
		}
	}
}
