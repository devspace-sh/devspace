package cmd

/*
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

/*type analyzeTestCase struct {
	name string

	fakeConfig           *latest.Config
	fakeKubeConfig       clientcmd.ClientConfig
	fakeKubeClient       kubectl.Client
	generatedYamlContent interface{}
	graphQLResponses     []interface{}
	providerList         []*cloudlatest.Provider
	waitFlag             bool
	globalFlags          flags.GlobalFlags

	expectedErr string
}

func TestAnalyze(t *testing.T) {
	testCases := []analyzeTestCase{
		analyzeTestCase{
			name: "Successful analysis with zero errors",
			globalFlags: flags.GlobalFlags{
				Namespace: "someNamespace",
			},
			fakeKubeClient: &kubectl.Client{
				Client: fake.NewSimpleClientset(),
			},
			fakeKubeConfig: &customKubeConfig{
				rawconfig: clientcmdapi.Config{
					Contexts: map[string]*clientcmdapi.Context{
						"": &clientcmdapi.Context{},
					},
					AuthInfos: map[string]*clientcmdapi.AuthInfo{
						"": &clientcmdapi.AuthInfo{},
					},
				},
			},
		},
	}

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
	assert.NilError(t, err, "Error getting homedir")
	componentDirBackup := filepath.Join(dir, "backup")
	err = fsutil.Copy(filepath.Join(homedir, generator.ComponentsRepoPath), componentDirBackup, false)
	assert.NilError(t, err, "Error creating a backup for the components")

	defer func() {
		err = os.RemoveAll(filepath.Join(homedir, generator.ComponentsRepoPath))
		assert.NilError(t, err, "Error removing component dir")
		err = fsutil.Copy(componentDirBackup, filepath.Join(homedir, generator.ComponentsRepoPath), false)
		assert.NilError(t, err, "Error restoring components")

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

	log.SetInstance(&log.DiscardLogger{PanicOnExit: true})

	for _, testCase := range testCases {
		testAnalyze(t, testCase)
	}
}

func testAnalyze(t *testing.T, testCase analyzeTestCase) {
	cloudpkg.DefaultGraphqlClient = &customGraphqlClient{
		responses: testCase.graphQLResponses,
	}

	loader.SetFakeConfig(testCase.fakeConfig)
	kubeconfig.SetFakeConfig(testCase.fakeKubeConfig)
	generated.ResetConfig()
	kubectl.SetFakeClient(testCase.fakeKubeClient)

	if testCase.generatedYamlContent != nil {
		content, err := yaml.Marshal(testCase.generatedYamlContent)
		assert.NilError(t, err, "Error parsing configs.yaml to yaml in testCase %s", testCase.name)
		fsutil.WriteToFile(content, generated.ConfigPath)
	}

	providerConfig, err := cloudconfig.Load()
	assert.NilError(t, err, "Error getting provider config in testCase %s", testCase.name)
	providerConfig.Providers = testCase.providerList

	err = (&AnalyzeCmd{
		GlobalFlags: &testCase.globalFlags,
		Wait:        testCase.waitFlag,
	}).RunAnalyze(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
}*/
