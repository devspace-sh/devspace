package list

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/cmd/flags"
	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	cloudlatest "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mgutz/ansi"

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

	expectedOutput string
	expectedErr    string
}

func TestListDeployments(t *testing.T) {
	expectedHeader := ansi.Color(" NAME  ", "green+b") + ansi.Color(" TYPE  ", "green+b") + ansi.Color(" DEPLOY  ", "green+b") + ansi.Color(" STATUS  ", "green+b")
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
					/*&latest.DeploymentConfig{
						Name: "ErrStatusHelm",
						Helm: &latest.HelmConfig{},
					},
					&latest.DeploymentConfig{
						Name:      "ErrStatusComponent",
						Component: &latest.ComponentConfig{},
					},*/
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
			expectedOutput: fmt.Sprintf("\nInfo Using kube context '%s'\nInfo Using namespace '%s'", ansi.Color("", "white+b"), ansi.Color("default", "white+b")) + "\nWarn Unable to create kubectl deploy config for UndeployableKubectl: No manifests defined for kubectl deploy\nWarn No deployment method defined for deployment NoDeploymentMethod\n" + expectedHeader + "\n No entries found\n\n",
		},
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testListDeployments(t, testCase)
	}
}

func testListDeployments(t *testing.T, testCase listDeploymentsTestCase) {
	logOutput = ""

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

	providerConfig, err := cloudconfig.ParseProviderConfig()
	assert.NilError(t, err, "Error getting provider config in testCase %s", testCase.name)
	providerConfig.Providers = testCase.providerList

	configutil.SetFakeConfig(testCase.fakeConfig)
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
	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
}
