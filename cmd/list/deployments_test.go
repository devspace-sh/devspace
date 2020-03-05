package list

/*
import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/devspace-cloud/devspace/cmd/flags"
	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	cloudlatest "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

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

type listDeploymentsTestCase struct {
	name string

	fakeConfig           *latest.Config
	fakeKubeConfig       clientcmd.ClientConfig
	configsYamlContent   interface{}
	generatedYamlContent interface{}
	providerList         []*cloudlatest.Provider

	expectTablePrint bool
	expectedHeader   []string
	expectedValues   [][]string
	expectedErr      string
}

func TestListDeployments(t *testing.T) {
	testCases := []listDeploymentsTestCase{
		listDeploymentsTestCase{
			name: "All deployments not listable",
			fakeConfig: &latest.Config{
				Deployments: []*latest.DeploymentConfig{
					&latest.DeploymentConfig{
						Name:    "UndeployableKubectl",
						Kubectl: &latest.KubectlConfig{},
					},
					//Those slow down the test
					&latest.DeploymentConfig{
						Name: "NoDeploymentMethod",
					},
				},
			},
			generatedYamlContent: generated.Config{},
			fakeKubeConfig: &customKubeConfig{
				rawconfig: clientcmdapi.Config{
					Contexts: map[string]*clientcmdapi.Context{
						"": &clientcmdapi.Context{},
					},
					Clusters: map[string]*clientcmdapi.Cluster{
						"": &clientcmdapi.Cluster{
							LocationOfOrigin: "someLocation",
							Server:           "someServer",
						},
					},
					AuthInfos: map[string]*clientcmdapi.AuthInfo{
						"": &clientcmdapi.AuthInfo{},
					},
				},
			},
			expectedHeader: []string{"NAME", "TYPE", "DEPLOY", "STATUS"},
		},
	}

	log.SetInstance(log.Discard)

	for _, testCase := range testCases {
		testListDeployments(t, testCase)
	}
}

func testListDeployments(t *testing.T, testCase listDeploymentsTestCase) {
	log.SetFakePrintTable(func(s log.Logger, header []string, values [][]string) {
		assert.Assert(t, testCase.expectTablePrint || len(testCase.expectedHeader)+len(testCase.expectedValues) > 0, "PrintTable unexpectedly called in testCase %s", testCase.name)
		assert.Equal(t, reflect.DeepEqual(header, testCase.expectedHeader), true, "Unexpected header in testCase %s. Expected:%v\nActual:%v", testCase.name, testCase.expectedHeader, header)
		assert.Equal(t, reflect.DeepEqual(values, testCase.expectedValues), true, "Unexpected values in testCase %s. Expected:%v\nActual:%v", testCase.name, testCase.expectedValues, values)
	})

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

	loader := cloudconfig.NewLoader()
	providerConfig, err := loader.Load()
	assert.NilError(t, err, "Error getting provider config in testCase %s", testCase.name)
	providerConfig.Providers = testCase.providerList

	loader.SetFakeConfig(testCase.fakeConfig)
	generated.ResetConfig()
	kubeconfig.SetFakeConfig(testCase.fakeKubeConfig)

	if testCase.generatedYamlContent != nil {
		content, err := yaml.Marshal(testCase.generatedYamlContent)
		assert.NilError(t, err, "Error parsing configs.yaml to yaml in testCase %s", testCase.name)
		fsutil.WriteToFile(content, generated.ConfigPath)
	}

	err = (&deploymentsCmd{GlobalFlags: &flags.GlobalFlags{}}).RunDeploymentsStatus(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
}*/
