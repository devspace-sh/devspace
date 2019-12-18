package cloud

import (
	"testing"
	"time"

	cloudclient "github.com/devspace-cloud/devspace/pkg/devspace/cloud/client"
	fakeclient "github.com/devspace-cloud/devspace/pkg/devspace/cloud/client/testing"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	fakekube "github.com/devspace-cloud/devspace/pkg/devspace/kubectl/testing"
	log "github.com/devspace-cloud/devspace/pkg/util/log/testing"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type defaultClusterSpaceDomainTestCase struct {
	name string

	client         kubectl.Client
	useHostNetwork bool
	clusterID      int
	key            string
	waitTimeout    time.Duration

	expectedErr    string
	expectedDomain string
}

func TestDefualtClusterSpaceDomain(t *testing.T) {
	clientWithEmptyNode := fake.NewSimpleClientset()
	clientWithEmptyNode.CoreV1().Nodes().Create(&corev1.Node{})
	clientWithPublicNode := fake.NewSimpleClientset()
	clientWithPublicNode.CoreV1().Nodes().Create(&corev1.Node{
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				corev1.NodeAddress{
					Type:    corev1.NodeExternalIP,
					Address: "someAddress",
				},
			},
		},
	})
	clientWithIngressHost := fake.NewSimpleClientset()
	clientWithIngressHost.CoreV1().Services(constants.DevSpaceCloudNamespace).Create(&corev1.Service{
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					corev1.LoadBalancerIngress{
						Hostname: "someHost",
					},
				},
			},
		},
	})
	clientWithIngressIP := fake.NewSimpleClientset()
	clientWithIngressIP.CoreV1().Services(constants.DevSpaceCloudNamespace).Create(&corev1.Service{
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					corev1.LoadBalancerIngress{
						IP: "someIP",
					},
				},
			},
		},
	})

	testCases := []defaultClusterSpaceDomainTestCase{
		defaultClusterSpaceDomainTestCase{
			name:           "no nodes",
			useHostNetwork: true,
			expectedErr:    "Couldn't find a node in cluster",
		},
		defaultClusterSpaceDomainTestCase{
			name:           "only one empty node",
			useHostNetwork: true,
			client: &fakekube.Client{
				Client: clientWithEmptyNode,
			},
			expectedErr: "Couldn't find a node with a valid external IP address in cluster, make sure your nodes are accessable from the outside",
		},
		defaultClusterSpaceDomainTestCase{
			name:           "find address in node",
			useHostNetwork: true,
			client: &fakekube.Client{
				Client: clientWithPublicNode,
			},
			expectedDomain: "default",
		},
		defaultClusterSpaceDomainTestCase{
			name:        "timeout",
			expectedErr: "Loadbalancer didn't receive a valid IP address in time. Skipping configuration of default domain for space subdomains",
		},
		defaultClusterSpaceDomainTestCase{
			name:        "Find ip in ingress host",
			waitTimeout: time.Second,
			client: &fakekube.Client{
				Client: clientWithIngressHost,
			},
			expectedDomain: "default",
		},
		defaultClusterSpaceDomainTestCase{
			name:        "Find ip in ingress IP",
			waitTimeout: time.Second,
			client: &fakekube.Client{
				Client: clientWithIngressIP,
			},
			expectedDomain: "default",
		},
	}

	waitTimeoutBackup := waitTimeout
	defer func() { waitTimeout = waitTimeoutBackup }()

	for _, testCase := range testCases {

		client := &fakeclient.CloudClient{
			Clusters: []*fakeclient.ClusterWithDomain{
				&fakeclient.ClusterWithDomain{
					Cluster: latest.Cluster{
						ClusterID: 0,
					},
				},
			},
		}
		provider := &provider{
			log:    &log.FakeLogger{},
			client: client,
		}

		if testCase.client == nil {
			testCase.client = &fakekube.Client{
				Client: fake.NewSimpleClientset(),
			}
		}

		waitTimeout = testCase.waitTimeout

		err := provider.defaultClusterSpaceDomain(testCase.client, testCase.useHostNetwork, testCase.clusterID, testCase.key)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error getting Key in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error from getKey in testCase %s", testCase.name)
		}
		assert.Equal(t, client.Clusters[0].Domain, testCase.expectedDomain, "Unexpected domain in testCase %s", testCase.name)
	}
}

type specifyDomainTestCase struct {
	name string

	clusterID int
	options   *ConnectClusterOptions
	answers   []string

	expectedErr    string
	expectedDomain string
}

func TestSpecifyDomain(t *testing.T) {
	testCases := []specifyDomainTestCase{
		specifyDomainTestCase{
			name:           "Update to answered domain",
			answers:        []string{"AnsweredDomain"},
			expectedDomain: "AnsweredDomain",
		},
	}

	for _, testCase := range testCases {
		logger := log.NewFakeLogger()
		for _, answer := range testCase.answers {
			logger.Survey.SetNextAnswer(answer)
		}

		client := &fakeclient.CloudClient{
			Clusters: []*fakeclient.ClusterWithDomain{
				&fakeclient.ClusterWithDomain{
					Cluster: latest.Cluster{
						ClusterID: 0,
					},
				},
			},
		}
		provider := &provider{
			log:    logger,
			client: client,
		}

		if testCase.options == nil {
			testCase.options = &ConnectClusterOptions{}
		}
		if testCase.options.UseHostNetwork == nil {
			testCase.options.UseHostNetwork = ptr.Bool(false)
		}

		err := provider.specifyDomain(0, testCase.options)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error getting Key in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error from getKey in testCase %s", testCase.name)
		}
		assert.Equal(t, client.Clusters[0].Domain, testCase.expectedDomain, "Unexpected domain in testCase %s", testCase.name)
	}
}

type needKeyTestCase struct {
	name string

	settings []cloudclient.Setting

	expectedErr  string
	expectedNeed bool
}

func TestNeedKey(t *testing.T) {
	testCases := []needKeyTestCase{
		needKeyTestCase{
			name: "no settings",
		},
		needKeyTestCase{
			name: "need is there",
			settings: []cloudclient.Setting{
				cloudclient.Setting{
					ID:    SettingDefaultClusterEncryptToken,
					Value: "true",
				},
			},
			expectedNeed: true,
		},
	}

	for _, testCase := range testCases {
		provider := &provider{
			client: &fakeclient.CloudClient{
				SettingsArr: testCase.settings,
			},
			log: &log.FakeLogger{},
		}

		returnedKey, err := provider.needKey()

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error getting Key in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error from getKey in testCase %s", testCase.name)
		}
		assert.Equal(t, returnedKey, testCase.expectedNeed, "Wrong need returned in testCase %s", testCase.name)
	}
}

type getServiceAccountCredentialsTestCase struct {
	name string

	kubeClient kubectl.Client
	timeout    time.Duration

	expectedErr   string
	expectedToken string
	expectedCert  string
}

func TestGetServiceAccountCredentials(t *testing.T) {
	clientWithUserSecret := fake.NewSimpleClientset()
	clientWithUserSecret.CoreV1().ServiceAccounts(DevSpaceCloudNamespace).Create(&corev1.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Name: DevSpaceServiceAccount,
		},
		Secrets: []corev1.ObjectReference{
			corev1.ObjectReference{
				Name: "mySecret",
			},
		},
	})
	clientWithUserSecret.CoreV1().Secrets(DevSpaceCloudNamespace).Create(&corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name: "mySecret",
		},
		Data: map[string][]byte{
			"token":  []byte("mytoken"),
			"ca.crt": []byte("1234"),
		},
	})

	testCases := []getServiceAccountCredentialsTestCase{
		getServiceAccountCredentialsTestCase{
			name: "timeout",
			kubeClient: &fakekube.Client{
				Client: clientWithUserSecret,
			},
			expectedErr: "ServiceAccount did not receive secret in time",
		},
		getServiceAccountCredentialsTestCase{
			name:    "get seret",
			timeout: time.Second,
			kubeClient: &fakekube.Client{
				Client: clientWithUserSecret,
			},
			expectedToken: "mytoken",
			expectedCert:  "MTIzNA==",
		},
	}

	waitTimeoutBackup := getServiceAccountTimeout
	defer func() {
		getServiceAccountTimeout = waitTimeoutBackup
	}()

	for _, testCase := range testCases {
		provider := &provider{
			log: &log.FakeLogger{},
		}

		if testCase.kubeClient == nil {
			testCase.kubeClient = &fakekube.Client{
				Client: fake.NewSimpleClientset(),
			}
		}

		getServiceAccountTimeout = testCase.timeout

		token, cert, err := provider.getServiceAccountCredentials(testCase.kubeClient)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error getting Key in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error from getKey in testCase %s", testCase.name)
		}
		assert.Equal(t, string(token), testCase.expectedToken, "Unexpected token in testCase %s", testCase.name)
		assert.Equal(t, cert, testCase.expectedCert, "Unexpected cert in testCase %s", testCase.name)
	}
}

type getKeyTestCase struct {
	name string

	givenKeys          map[int]string
	forceQuestionParam bool
	answers            []string

	expectedErr string
	expectedKey string
}

func TestGetKey(t *testing.T) {
	testCases := []getKeyTestCase{
		getKeyTestCase{
			name:               "One key, no question",
			givenKeys:          map[int]string{5: "onlyKey"},
			forceQuestionParam: false,
			expectedKey:        "onlyKey",
		},
		getKeyTestCase{
			name:               "Key from question",
			forceQuestionParam: true,
			answers:            []string{"firstKey", "secondKey", "sameKey", "sameKey"},
			expectedKey:        "716fb307cf5cc64f34acfe748560a1a268d6e1a47d56ff1fc64eb549bcecd3f1",
		},
	}

	for _, testCase := range testCases {
		logger := log.NewFakeLogger()
		for _, answer := range testCase.answers {
			logger.Survey.SetNextAnswer(answer)
		}
		provider := &provider{
			Provider: latest.Provider{
				ClusterKey: testCase.givenKeys,
			},
			log: logger,
		}

		returnedKey, err := provider.getKey(testCase.forceQuestionParam)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error getting Key in testCase %s", testCase.name)
			assert.Equal(t, returnedKey, testCase.expectedKey, "Wrong key returned in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error from getKey in testCase %s", testCase.name)
		}
	}
}

type getClusternameTestCase struct {
	name string

	clusterName string
	answers     []string

	expectedErr         string
	expectedClustername string
}

func TestGetClustername(t *testing.T) {
	testCases := []getClusternameTestCase{
		getClusternameTestCase{
			name:        "Invalid clustername",
			clusterName: "%",
			expectedErr: "Cluster name % can only contain letters, numbers and dashes (-)",
		},
		getClusternameTestCase{
			name:                "Valid clustername",
			clusterName:         "valid-name-1",
			expectedClustername: "valid-name-1",
		},
		getClusternameTestCase{
			name:                "Clustername from question",
			answers:             []string{"()", "valid-name-2"},
			expectedClustername: "valid-name-2",
		},
	}

	for _, testCase := range testCases {
		logger := log.NewFakeLogger()
		for _, answer := range testCase.answers {
			logger.Survey.SetNextAnswer(answer)
		}
		provider := &provider{
			log: logger,
		}

		clusterName, err := provider.getClusterName(testCase.clusterName)

		assert.Equal(t, clusterName, testCase.expectedClustername, "Wrong key returned in testCase %s", testCase.name)
		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}
	}
}

// Kubectl fake client still missing
/*type checkResourcesTestCase struct {
	name         string
	provider     *provider
	createdNodes []*k8sv1.Node

	expectedErr string
}

func TestCheckResources(t *testing.T) {
	testCases := []checkResourcesTestCase{}

	for _, testCase := range testCases {
		kubeClient := fake.NewSimpleClientset()
		for _, node := range testCase.createdNodes {
			kubeClient.CoreV1().Nodes().Create(node)
		}

		_, err := testCase.provider.checkResources(kubeClient)
		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error checking resources in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error from checking resources in testCase %s", testCase.name)
		}
	}
}*/

type initializeNamespaceTestCase struct {
	name string

	client kubernetes.Interface

	expectedErr                 string
	expectedNamespaces          interface{}
	expectedServiceAccounts     interface{}
	expectedClusterRoleBindings interface{}
}

func TestInitializeNamespace(t *testing.T) {
	testCases := []initializeNamespaceTestCase{
		initializeNamespaceTestCase{
			name:   "Init namespace",
			client: fake.NewSimpleClientset(),
			expectedNamespaces: &corev1.NamespaceList{
				Items: []corev1.Namespace{
					corev1.Namespace{
						ObjectMeta: v1.ObjectMeta{
							Name: "devspace-cloud",
							Labels: map[string]string{
								"devspace.cloud/control-plane": "true",
							},
						},
					},
				},
			},
			expectedServiceAccounts: &corev1.ServiceAccountList{
				Items: []corev1.ServiceAccount{
					corev1.ServiceAccount{
						ObjectMeta: v1.ObjectMeta{
							Name:      "devspace-cloud-user",
							Namespace: "devspace-cloud",
						},
					},
				},
			},
			expectedClusterRoleBindings: &rbacv1.ClusterRoleBindingList{
				Items: []rbacv1.ClusterRoleBinding{
					rbacv1.ClusterRoleBinding{
						ObjectMeta: v1.ObjectMeta{
							Name: "devspace-cloud-user-binding",
						},
						Subjects: []rbacv1.Subject{
							rbacv1.Subject{
								Kind:      "ServiceAccount",
								Name:      "devspace-cloud-user",
								Namespace: "devspace-cloud",
							},
						},
						RoleRef: rbacv1.RoleRef{
							APIGroup: "rbac.authorization.k8s.io",
							Kind:     "ClusterRole",
							Name:     "cluster-admin",
						},
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		provider := &provider{
			log: &log.FakeLogger{},
		}

		client := testCase.client
		err := provider.initializeNamespace(client)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		namespaces, err := client.CoreV1().Namespaces().List(v1.ListOptions{})
		assert.NilError(t, err, "Error listing namespaces in testCase %s", testCase.name)
		namespacesAsYaml, err := yaml.Marshal(namespaces)
		assert.NilError(t, err, "Error parsing namespaces in testCase %s", testCase.name)
		expectedNamespacesAsYaml, err := yaml.Marshal(testCase.expectedNamespaces)
		assert.NilError(t, err, "Error parsing expected namespaces in testCase %s", testCase.name)
		assert.Equal(t, string(namespacesAsYaml), string(expectedNamespacesAsYaml), "Unexpected namespaces in testCase %s", testCase.name)

		serviceAccounts, err := client.CoreV1().ServiceAccounts(DevSpaceCloudNamespace).List(v1.ListOptions{})
		assert.NilError(t, err, "Error listing serviceAccounts in testCase %s", testCase.name)
		serviceAccountsAsYaml, err := yaml.Marshal(serviceAccounts)
		assert.NilError(t, err, "Error parsing serviceAccounts in testCase %s", testCase.name)
		expectedServiceAccountsAsYaml, err := yaml.Marshal(testCase.expectedServiceAccounts)
		assert.NilError(t, err, "Error parsing expected serviceAccounts in testCase %s", testCase.name)
		assert.Equal(t, string(serviceAccountsAsYaml), string(expectedServiceAccountsAsYaml), "Unexpected serviceAccounts in testCase %s", testCase.name)

		clusterRoleBindings, err := client.RbacV1().ClusterRoleBindings().List(v1.ListOptions{})
		assert.NilError(t, err, "Error listing clusterRoleBindings in testCase %s", testCase.name)
		clusterRoleBindingsAsYaml, err := yaml.Marshal(clusterRoleBindings)
		assert.NilError(t, err, "Error parsing clusterRoleBindings in testCase %s", testCase.name)
		expectedClusterRoleBindingsAsYaml, err := yaml.Marshal(testCase.expectedClusterRoleBindings)
		assert.NilError(t, err, "Error parsing expected clusterRoleBindings in testCase %s", testCase.name)
		assert.Equal(t, string(clusterRoleBindingsAsYaml), string(expectedClusterRoleBindingsAsYaml), "Unexpected clusterRoleBindings in testCase %s", testCase.name)
	}
}
