package reset

import (
	"bytes"
	"encoding/json"
	"fmt"

	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"

	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type customGraphqlClient struct {
	responses []interface{}
}

func (q *customGraphqlClient) GrapqhlRequest(p *cloudpkg.Provider, request string, vars map[string]interface{}, response interface{}) error {
	if len(q.responses) == 0 {
		panic("Not enough responses. Need response for: " + request)
	}
	currentResponse := q.responses[0]
	q.responses = q.responses[1:]

	errorResponse, isError := currentResponse.(error)
	if isError {
		return errorResponse
	}
	buf, err := json.Marshal(currentResponse)
	if err != nil {
		panic(fmt.Sprintf("Cannot encode response. %d responses left", len(q.responses)))
	}
	json.NewDecoder(bytes.NewReader(buf)).Decode(&response)

	return nil
}

type customKubeConfig struct {
	rawconfig      clientcmdapi.Config
	rawConfigError error

	clientConfig      *restclient.Config
	clientConfigError error

	namespace     string
	namespaceBool bool
	namespaceErr  error

	configAccess clientcmd.ConfigAccess
}

func (config *customKubeConfig) RawConfig() (clientcmdapi.Config, error) {
	return config.rawconfig, config.rawConfigError
}
func (config *customKubeConfig) Namespace() (string, bool, error) {
	return config.namespace, config.namespaceBool, config.namespaceErr
}
func (config *customKubeConfig) ClientConfig() (*restclient.Config, error) {
	return config.clientConfig, config.clientConfigError
}
func (config *customKubeConfig) ConfigAccess() clientcmd.ConfigAccess {
	return config.configAccess
}

/*type resetKeyTestCase struct {
	name string

	args             []string
	answers          []string
	graphQLResponses []interface{}
	provider         string
	providerList     []*cloudlatest.Provider
	fakeKubeConfig   clientcmd.ClientConfig
	fakeKubeClient   kubectl.Client

	expectedErr string
}

func TestRunResetKey(t *testing.T) {
	claimAsJSON, _ := json.Marshal(token.ClaimSet{
		Expiration: time.Now().Add(time.Hour).Unix(),
		Hasura: token.Hasura{
			AccountID: "1",
		},
	})
	validEncodedClaim := base64.URLEncoding.EncodeToString(claimAsJSON)
	for strings.HasSuffix(string(validEncodedClaim), "=") {
		validEncodedClaim = strings.TrimSuffix(validEncodedClaim, "=")
	}

	kubeClientWithDSCloudUser := fake.NewSimpleClientset()
	_, err := kubeClientWithDSCloudUser.CoreV1().ServiceAccounts(cloudpkg.DevSpaceCloudNamespace).Create(&v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: "devspace-cloud-user",
		},
		Secrets: []v1.ObjectReference{
			v1.ObjectReference{},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = kubeClientWithDSCloudUser.CoreV1().Secrets(cloudpkg.DevSpaceCloudNamespace).Create(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	testCases := []resetKeyTestCase{
		resetKeyTestCase{
			name:     "Successful reset",
			provider: "myProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name:  "myProvider",
					Key:   "someKey",
					Token: "." + validEncodedClaim + ".",
				},
			},
			args: []string{"myCluster"},
			graphQLResponses: []interface{}{
				struct {
					Clusters []*cloudlatest.Cluster `json:"cluster"`
				}{
					Clusters: []*cloudlatest.Cluster{
						&cloudlatest.Cluster{
							Name:   "myCluster",
							Server: ptr.String(""),
						},
					},
				},
				struct {
					ClusterUser []*cloudlatest.ClusterUser `json:"cluster_user"`
				}{
					ClusterUser: []*cloudlatest.ClusterUser{
						&cloudlatest.ClusterUser{},
					},
				},
				struct{}{},
			},
			fakeKubeConfig: &customKubeConfig{
				rawconfig: clientcmdapi.Config{
					Contexts: map[string]*clientcmdapi.Context{
						"": &clientcmdapi.Context{},
					},
					AuthInfos: map[string]*clientcmdapi.AuthInfo{
						"": &clientcmdapi.AuthInfo{},
					},
					Clusters: map[string]*clientcmdapi.Cluster{
						"": &clientcmdapi.Cluster{},
					},
				},
			},
			fakeKubeClient: &kubectl.Client{
				Client:     kubeClientWithDSCloudUser,
				RestConfig: &restclient.Config{},
			},
			answers: []string{"", "encryptionKey", "encryptionKey"},
		},
	}

	log.SetInstance(&log.DiscardLogger{PanicOnExit: true})

	for _, testCase := range testCases {
		testRunResetKey(t, testCase)
	}
}

func testRunResetKey(t *testing.T, testCase resetKeyTestCase) {
	dir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}

	homedir, err := homedir.Dir()
	assert.NilError(t, err, "Error getting homedir in testCase %s", testCase.name)
	relDir, err := filepath.Rel(homedir, dir)
	assert.NilError(t, err, "Error getting relative dir path in testCase %s", testCase.name)
	cloudconfig.DevSpaceProvidersConfigPath = filepath.Join(relDir, "Doesn'tExist")
	cloudconfig.LegacyDevSpaceCloudConfigPath = filepath.Join(relDir, "Doesn'tExist")

	providerConfig, err := cloudconfig.Load()
	assert.NilError(t, err, "Error getting provider config in testCase %s", testCase.name)
	providerConfig.Providers = testCase.providerList

	kubeconfig.SetFakeConfig(testCase.fakeKubeConfig)
	kubectl.SetFakeClient(testCase.fakeKubeClient)

	for _, answer := range testCase.answers {
		survey.SetNextAnswer(answer)
	}

	cloudpkg.DefaultGraphqlClient = &customGraphqlClient{
		responses: testCase.graphQLResponses,
	}

	defer func() {
		//Delete temp folder
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}()

	if len(testCase.args) == 0 {
		testCase.args = []string{""}
	}
	err = (&keyCmd{
		Provider: testCase.provider,
	}).RunResetkey(nil, testCase.args)

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
}*/
