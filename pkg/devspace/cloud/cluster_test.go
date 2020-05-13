package cloud

import (
	"context"
	"testing"
	"time"

	cloudclient "github.com/devspace-cloud/devspace/pkg/devspace/cloud/client"
	fakeclient "github.com/devspace-cloud/devspace/pkg/devspace/cloud/client/testing"
	testconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/testing"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	fakekube "github.com/devspace-cloud/devspace/pkg/devspace/kubectl/testing"
	fakeBrowser "github.com/devspace-cloud/devspace/pkg/util/browser/testing"
	log "github.com/devspace-cloud/devspace/pkg/util/log/testing"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	fakesurvey "github.com/devspace-cloud/devspace/pkg/util/survey/testing"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type connectClusterTestCase struct {
	name string

	options         *ConnectClusterOptions
	nodes           []*corev1.Node
	services        []*corev1.Service
	serviceAccounts []*corev1.ServiceAccount
	secrets         []*corev1.Secret
	client          fakeclient.CloudClient
	answers         []string

	expectedErr         string
	expectedClientState interface{}
}

func TestConnectCluster(t *testing.T) {
	testCases := []connectClusterTestCase{
		connectClusterTestCase{
			name: "Successfully connect private cluster",
			options: &ConnectClusterOptions{
				ClusterName:             "myCluster",
				UseDomain:               true,
				DeployIngressController: true,
				OpenUI:                  true,
			},
			nodes: []*corev1.Node{&corev1.Node{}},
			serviceAccounts: []*corev1.ServiceAccount{
				&corev1.ServiceAccount{
					ObjectMeta: v1.ObjectMeta{
						Name: DevSpaceServiceAccount,
					},
					Secrets: []corev1.ObjectReference{
						corev1.ObjectReference{
							Name: "mySecret",
						},
					},
				},
			},
			secrets: []*corev1.Secret{
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name: "mySecret",
					},
					Data: map[string][]byte{
						"token":  []byte("mytoken"),
						"ca.crt": []byte("1234"),
					},
				},
			},
			client: fakeclient.CloudClient{
				SettingsArr: []cloudclient.Setting{
					cloudclient.Setting{
						ID:    SettingDefaultClusterEncryptToken,
						Value: "true",
					},
				},
				ClusterKeys: map[int]string{0: "3af21320d022362b98b60808b0e012ef3a1d696e04760018ac40d4cf3ef27c85"},
			},
			answers: []string{"typedKey", "typedKey", hostNetworkOption, "someHost"},
			expectedClientState: fakeclient.CloudClient{
				Clusters: []*fakeclient.ExtendedCluster{
					&fakeclient.ExtendedCluster{
						Cluster: latest.Cluster{
							Server:       ptr.String("HostNetwork"),
							Name:         "myCluster",
							EncryptToken: true,
						},
						Domain:   "someHost",
						Deployed: []string{"IngressController"},
					},
				},
				ClusterKeys: map[int]string{0: "3af21320d022362b98b60808b0e012ef3a1d696e04760018ac40d4cf3ef27c85"},
				SettingsArr: []cloudclient.Setting{
					cloudclient.Setting{
						ID:    SettingDefaultClusterEncryptToken,
						Value: "true",
					},
				},
			},
		},
		connectClusterTestCase{
			name: "Successfully connect public cluster",
			options: &ConnectClusterOptions{
				ClusterName:             "pubCluster",
				Public:                  true,
				DeployIngressController: true,
			},
			nodes: []*corev1.Node{&corev1.Node{}},
			services: []*corev1.Service{
				&corev1.Service{
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
				},
			},
			serviceAccounts: []*corev1.ServiceAccount{
				&corev1.ServiceAccount{
					ObjectMeta: v1.ObjectMeta{
						Name: DevSpaceServiceAccount,
					},
					Secrets: []corev1.ObjectReference{
						corev1.ObjectReference{
							Name: "mySecret",
						},
					},
				},
			},
			secrets: []*corev1.Secret{
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name: "mySecret",
					},
					Data: map[string][]byte{
						"token":  []byte("mytoken"),
						"ca.crt": []byte("1234"),
					},
				},
			},
			client: fakeclient.CloudClient{
				SettingsArr: []cloudclient.Setting{
					cloudclient.Setting{
						ID:    SettingDefaultClusterEncryptToken,
						Value: "true",
					},
				},
			},
			//answers: []string{"typedKey", "typedKey", hostNetworkOption, "someHost"},
			expectedClientState: fakeclient.CloudClient{
				Clusters: []*fakeclient.ExtendedCluster{
					&fakeclient.ExtendedCluster{
						Cluster: latest.Cluster{
							Server: ptr.String("testHost"),
							Name:   "pubCluster",
						},
						Domain:   "default",
						Deployed: []string{"IngressController"},
					},
				},
				SettingsArr: []cloudclient.Setting{
					cloudclient.Setting{
						ID:    SettingDefaultClusterEncryptToken,
						Value: "true",
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		kube := &fakekube.FakeFakeClientset{
			Clientset:   *fake.NewSimpleClientset(),
			RBACEnabled: true,
		}
		for _, node := range testCase.nodes {
			kube.CoreV1().Nodes().Create(context.TODO(), node, v1.CreateOptions{})
		}
		for _, service := range testCase.services {
			kube.CoreV1().Services(DevSpaceCloudNamespace).Create(context.TODO(), service, v1.CreateOptions{})
		}
		for _, sa := range testCase.serviceAccounts {
			kube.CoreV1().ServiceAccounts(DevSpaceCloudNamespace).Create(context.TODO(), sa, v1.CreateOptions{})
		}
		for _, secret := range testCase.secrets {
			kube.CoreV1().Secrets(DevSpaceCloudNamespace).Create(context.TODO(), secret, v1.CreateOptions{})
		}
		kubeClient := &fakekube.Client{
			Client: kube,
		}

		logger := log.NewFakeLogger()
		for _, answer := range testCase.answers {
			logger.Survey.SetNextAnswer(answer)
		}

		provider := &provider{
			Provider: latest.Provider{
				ClusterKey: map[int]string{},
			},
			log:        logger,
			kubeClient: kubeClient,
			client:     &testCase.client,
			loader:     testconfig.NewLoader(&latest.Config{}),
			browser: &fakeBrowser.FakeBrowser{
				StartCallback: func(url string) error { return errors.New("") },
			},
		}

		if testCase.options == nil {
			testCase.options = &ConnectClusterOptions{}
		}

		err := provider.ConnectCluster(testCase.options)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		clientAsYaml, err := yaml.Marshal(provider.client)
		assert.NilError(t, err, "Error parsing client to yaml in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedClientState)
		assert.NilError(t, err, "Error parsing client expection to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(clientAsYaml), string(expectedAsYaml), "Unexpected client state in testCase %s", testCase.name)
	}
}

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
	clientWithEmptyNode.CoreV1().Nodes().Create(context.TODO(), &corev1.Node{}, v1.CreateOptions{})
	clientWithPublicNode := fake.NewSimpleClientset()
	clientWithPublicNode.CoreV1().Nodes().Create(context.TODO(), &corev1.Node{
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				corev1.NodeAddress{
					Type:    corev1.NodeExternalIP,
					Address: "someAddress",
				},
			},
		},
	}, v1.CreateOptions{})
	clientWithIngressHost := fake.NewSimpleClientset()
	clientWithIngressHost.CoreV1().Services(constants.DevSpaceCloudNamespace).Create(context.TODO(), &corev1.Service{
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
	}, v1.CreateOptions{})
	clientWithIngressIP := fake.NewSimpleClientset()
	clientWithIngressIP.CoreV1().Services(constants.DevSpaceCloudNamespace).Create(context.TODO(), &corev1.Service{
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
	}, v1.CreateOptions{})

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
			Clusters: []*fakeclient.ExtendedCluster{
				&fakeclient.ExtendedCluster{
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
			Clusters: []*fakeclient.ExtendedCluster{
				&fakeclient.ExtendedCluster{
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

type deployServicesTestCase struct {
	name string

	answers     []string
	configMaps  []corev1.ConfigMap
	certManager bool
	options     *ConnectClusterOptions

	expectedErr                       string
	expectDeployedIngressController   bool
	expectDeployedAdmissionController bool
	expectDeployedGatekeeper          bool
	expectDeployedGatekeeperRules     bool
	expectDeployedCertManager         bool
	expectHostNetwork                 bool
}

func TestDeployServices(t *testing.T) {
	testCases := []deployServicesTestCase{
		deployServicesTestCase{
			name: "Deploy nothing",
			configMaps: []corev1.ConfigMap{
				corev1.ConfigMap{
					ObjectMeta: v1.ObjectMeta{
						Name:   "someConfigMap",
						Labels: map[string]string{"NAME": "devspace-cloud", "OWNER": "TILLER", "STATUS": "DEPLOYED"},
					},
				},
			},
			options: &ConnectClusterOptions{
				DeployIngressController: true,
			},
		},
		deployServicesTestCase{
			name:    "Deploy everything",
			answers: []string{hostNetworkOption},
			options: &ConnectClusterOptions{
				DeployIngressController:   true,
				DeployAdmissionController: true,
				DeployGatekeeper:          true,
				DeployGatekeeperRules:     true,
				DeployCertManager:         true,
			},
			expectDeployedIngressController:   true,
			expectDeployedAdmissionController: true,
			expectDeployedGatekeeper:          true,
			expectDeployedGatekeeperRules:     true,
			expectDeployedCertManager:         true,
			expectHostNetwork:                 true,
		},
	}

	for _, testCase := range testCases {
		kube := fake.NewSimpleClientset()
		for _, configMap := range testCase.configMaps {
			kube.CoreV1().ConfigMaps(DevSpaceCloudNamespace).Create(context.TODO(), &configMap, v1.CreateOptions{})
		}
		kubeClient := &fakekube.Client{
			Client: kube,
		}

		client := &fakeclient.CloudClient{
			Clusters: []*fakeclient.ExtendedCluster{
				&fakeclient.ExtendedCluster{
					Cluster: latest.Cluster{
						ClusterID: 0,
					},
				},
			},
		}

		logger := &log.FakeLogger{
			Survey: &fakesurvey.FakeSurvey{},
		}
		provider := &provider{
			client:     client,
			log:        logger,
			kubeClient: kubeClient,
		}

		for _, answer := range testCase.answers {
			logger.Survey.SetNextAnswer(answer)
		}

		if testCase.options == nil {
			testCase.options = &ConnectClusterOptions{}
		}

		err := provider.deployServices(0, &clusterResources{CertManager: testCase.certManager}, testCase.options)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error getting Key in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error from getKey in testCase %s", testCase.name)
		}

		deployedList := client.Clusters[0].Deployed
		assert.Equal(t, testCase.expectDeployedIngressController, includes(deployedList, "IngressController"), "IngressController unexpectedly deployed or undeployed in testCase %s", testCase.name)
		assert.Equal(t, testCase.expectDeployedAdmissionController, includes(deployedList, "AdmissionController"), "AdmissionController unexpectedly deployed or undeployed in testCase %s", testCase.name)
		assert.Equal(t, testCase.expectDeployedGatekeeper, includes(deployedList, "Gatekeeper"), "Gatekeeper unexpectedly deployed or undeployed in testCase %s", testCase.name)
		assert.Equal(t, testCase.expectDeployedGatekeeperRules, includes(deployedList, "GatekeeperRules"), "GatekeeperRules unexpectedly deployed or undeployed in testCase %s", testCase.name)
		assert.Equal(t, testCase.expectDeployedCertManager, includes(deployedList, "CertManager"), "CertManager unexpectedly deployed or undeployed in testCase %s", testCase.name)
		assert.Equal(t, testCase.expectHostNetwork, client.Clusters[0].Cluster.Server != nil && *client.Clusters[0].Cluster.Server == "HostNetwork", "HostNetwork unexpectedly used or not used in testCase %s", testCase.name)
	}
}

func includes(haystack []string, needle string) bool {
	for _, subject := range haystack {
		if subject == needle {
			return true
		}
	}
	return false
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
	clientWithUserSecret.CoreV1().ServiceAccounts(DevSpaceCloudNamespace).Create(context.TODO(), &corev1.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Name: DevSpaceServiceAccount,
		},
		Secrets: []corev1.ObjectReference{
			corev1.ObjectReference{
				Name: "mySecret",
			},
		},
	}, v1.CreateOptions{})
	clientWithUserSecret.CoreV1().Secrets(DevSpaceCloudNamespace).Create(context.TODO(), &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name: "mySecret",
		},
		Data: map[string][]byte{
			"token":  []byte("mytoken"),
			"ca.crt": []byte("1234"),
		},
	}, v1.CreateOptions{})

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
			log:        &log.FakeLogger{},
			kubeClient: testCase.kubeClient,
		}

		if testCase.kubeClient == nil {
			testCase.kubeClient = &fakekube.Client{
				Client: fake.NewSimpleClientset(),
			}
		}

		getServiceAccountTimeout = testCase.timeout

		token, cert, err := provider.getServiceAccountCredentials()

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
type checkResourcesTestCase struct {
	name string

	createdNodes []*corev1.Node
	RBACEnabled  bool

	expectedErr string
}

func TestCheckResources(t *testing.T) {
	testCases := []checkResourcesTestCase{
		checkResourcesTestCase{
			name:        "No nodes",
			expectedErr: "The cluster specified has no nodes, please choose a cluster where at least one node is up and running",
		},
		checkResourcesTestCase{
			name: "No RBAC",
			createdNodes: []*corev1.Node{
				&corev1.Node{},
			},
			expectedErr: "Group version rbac.authorization.k8s.io/v1beta1 does not exist in cluster, but is required. Is RBAC enabled?",
		},
		checkResourcesTestCase{
			name: "Successful run",
			createdNodes: []*corev1.Node{
				&corev1.Node{},
			},
			RBACEnabled: true,
		},
	}

	for _, testCase := range testCases {
		kube := &fakekube.FakeFakeClientset{
			Clientset:   *fake.NewSimpleClientset(),
			RBACEnabled: testCase.RBACEnabled,
		}
		for _, node := range testCase.createdNodes {
			kube.CoreV1().Nodes().Create(context.TODO(), node, v1.CreateOptions{})
		}

		provider := &provider{
			log: &log.FakeLogger{},
			kubeClient: &fakekube.Client{
				Client: kube,
			},
		}

		_, err := provider.checkResources()

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error checking resources in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error from checking resources in testCase %s", testCase.name)
		}
	}
}

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
			kubeClient: &fakekube.Client{
				Client: testCase.client,
			},
		}

		err := provider.initializeNamespace()

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		namespaces, err := testCase.client.CoreV1().Namespaces().List(context.TODO(), v1.ListOptions{})
		assert.NilError(t, err, "Error listing namespaces in testCase %s", testCase.name)
		namespacesAsYaml, err := yaml.Marshal(namespaces)
		assert.NilError(t, err, "Error parsing namespaces in testCase %s", testCase.name)
		expectedNamespacesAsYaml, err := yaml.Marshal(testCase.expectedNamespaces)
		assert.NilError(t, err, "Error parsing expected namespaces in testCase %s", testCase.name)
		assert.Equal(t, string(namespacesAsYaml), string(expectedNamespacesAsYaml), "Unexpected namespaces in testCase %s", testCase.name)

		serviceAccounts, err := testCase.client.CoreV1().ServiceAccounts(DevSpaceCloudNamespace).List(context.TODO(), v1.ListOptions{})
		assert.NilError(t, err, "Error listing serviceAccounts in testCase %s", testCase.name)
		serviceAccountsAsYaml, err := yaml.Marshal(serviceAccounts)
		assert.NilError(t, err, "Error parsing serviceAccounts in testCase %s", testCase.name)
		expectedServiceAccountsAsYaml, err := yaml.Marshal(testCase.expectedServiceAccounts)
		assert.NilError(t, err, "Error parsing expected serviceAccounts in testCase %s", testCase.name)
		assert.Equal(t, string(serviceAccountsAsYaml), string(expectedServiceAccountsAsYaml), "Unexpected serviceAccounts in testCase %s", testCase.name)

		clusterRoleBindings, err := testCase.client.RbacV1().ClusterRoleBindings().List(context.TODO(), v1.ListOptions{})
		assert.NilError(t, err, "Error listing clusterRoleBindings in testCase %s", testCase.name)
		clusterRoleBindingsAsYaml, err := yaml.Marshal(clusterRoleBindings)
		assert.NilError(t, err, "Error parsing clusterRoleBindings in testCase %s", testCase.name)
		expectedClusterRoleBindingsAsYaml, err := yaml.Marshal(testCase.expectedClusterRoleBindings)
		assert.NilError(t, err, "Error parsing expected clusterRoleBindings in testCase %s", testCase.name)
		assert.Equal(t, string(clusterRoleBindingsAsYaml), string(expectedClusterRoleBindingsAsYaml), "Unexpected clusterRoleBindings in testCase %s", testCase.name)
	}
}

type resetKeyTestCase struct {
	name string

	clusterName     string
	clusters        []*fakeclient.ExtendedCluster
	answers         []string
	serviceAccounts []corev1.ServiceAccount
	secrets         []corev1.Secret

	expectedErr string
	expectedKey string
}

func TestResetKey(t *testing.T) {
	testCases := []resetKeyTestCase{
		resetKeyTestCase{
			name:        "Wrong host",
			clusterName: "testCluster",
			clusters: []*fakeclient.ExtendedCluster{
				&fakeclient.ExtendedCluster{
					Cluster: latest.Cluster{
						Name:   "testCluster",
						Server: ptr.String("clusterServer"),
					},
				},
			},
			expectedErr: "Selected context does not point to the correct host. Selected testHost <> clusterServer",
		},
		resetKeyTestCase{
			name:        "Successful reset",
			clusterName: "testCluster",
			clusters: []*fakeclient.ExtendedCluster{
				&fakeclient.ExtendedCluster{
					Cluster: latest.Cluster{
						ClusterID: 0,
						Name:      "testCluster",
						Server:    ptr.String("testHost"),
					},
				},
			},
			answers: []string{"validKey", "validKey"},
			serviceAccounts: []corev1.ServiceAccount{
				corev1.ServiceAccount{
					ObjectMeta: v1.ObjectMeta{
						Name: DevSpaceServiceAccount,
					},
					Secrets: []corev1.ObjectReference{
						corev1.ObjectReference{
							Name: "mySecret",
						},
					},
				},
			},
			secrets: []corev1.Secret{
				corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name: "mySecret",
					},
					Data: map[string][]byte{
						"token":  []byte("mytoken"),
						"ca.crt": []byte("1234"),
					},
				},
			},
			expectedKey: "94363a3c5e8f9f6d18dde3fbff991cb88a1f1ee1b5d8a47ca7cd2b2dcf66be1d",
		},
	}

	for _, testCase := range testCases {
		logger := log.NewFakeLogger()
		for _, answer := range testCase.answers {
			logger.Survey.SetNextAnswer(answer)
		}

		kubeClient := fake.NewSimpleClientset()
		for _, sa := range testCase.serviceAccounts {
			kubeClient.CoreV1().ServiceAccounts(DevSpaceCloudNamespace).Create(context.TODO(), &sa, v1.CreateOptions{})
		}
		for _, secret := range testCase.secrets {
			kubeClient.CoreV1().Secrets(DevSpaceCloudNamespace).Create(context.TODO(), &secret, v1.CreateOptions{})
		}

		provider := &provider{
			Provider: latest.Provider{
				ClusterKey: map[int]string{},
			},
			client: &fakeclient.CloudClient{
				Clusters: testCase.clusters,
			},
			kubeClient: &fakekube.Client{
				Client: kubeClient,
			},
			log:    logger,
			loader: testconfig.NewLoader(&latest.Config{}),
		}

		err := provider.ResetKey(testCase.clusterName)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		assert.Equal(t, testCase.expectedKey, provider.ClusterKey[0], "Wrong clusterKey in testCase %s", testCase.name)
	}
}
