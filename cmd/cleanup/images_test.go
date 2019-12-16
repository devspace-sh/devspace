package cleanup

/*
import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

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

type RunCleanupImagesTestCase struct {
	name string

	fakeConfig     *latest.Config
	fakeKubeConfig clientcmd.ClientConfig
	files          map[string]interface{}
	globalFlags    flags.GlobalFlags

	answers []string

	expectedErr string
}

func TestRunCleanupImages(t *testing.T) {
	testCases := []RunCleanupImagesTestCase{
		RunCleanupImagesTestCase{
			name:       "No images to delete",
			fakeConfig: &latest.Config{},
		},
		RunCleanupImagesTestCase{
			name: "Error getting kube config",
			fakeConfig: &latest.Config{
				Images: map[string]*latest.ImageConfig{
					"imageToDelete": &latest.ImageConfig{
						Image: "imageToDelete",
					},
				},
			},
			fakeKubeConfig: &customKubeConfig{
				rawConfigError: fmt.Errorf("RawConfigError"),
			},
			expectedErr: "RawConfigError",
		},
		RunCleanupImagesTestCase{
			name: "One image to delete",
			fakeConfig: &latest.Config{
				Images: map[string]*latest.ImageConfig{
					"imageToDelete": &latest.ImageConfig{
						Image: "imageToDelete",
					},
				},
			},
			globalFlags: flags.GlobalFlags{
				KubeContext: "someKubeContext",
			},
			fakeKubeConfig: &customKubeConfig{},
		},
	}

	for _, testCase := range testCases {
		testRunCleanupImages(t, testCase)
	}

}

func testRunCleanupImages(t *testing.T, testCase RunCleanupImagesTestCase) {
	dir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}

	for _, answer := range testCase.answers {
		survey.SetNextAnswer(answer)
	}

	kubeconfig.SetFakeConfig(testCase.fakeKubeConfig)
	isDeploymentsNil := testCase.fakeConfig == nil || testCase.fakeConfig.Deployments == nil
	if testCase.fakeConfig == nil {
		loader.ResetConfig()
	} else {
		loader.SetFakeConfig(testCase.fakeConfig)
		if isDeploymentsNil && testCase.fakeConfig != nil {
			testCase.fakeConfig.Deployments = nil
		}
	}

	for path, content := range testCase.files {
		asYAML, err := yaml.Marshal(content)
		assert.NilError(t, err, "Error parsing config to yaml in testCase %s", testCase.name)
		err = fsutil.WriteToFile(asYAML, path)
		assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
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

	log.SetInstance(&log.DiscardLogger{PanicOnExit: true})

	err = (&imagesCmd{GlobalFlags: &testCase.globalFlags}).RunCleanupImages(nil, nil)

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
}*/
