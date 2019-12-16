package remove

/*import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	cloudlatest "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	homedir "github.com/mitchellh/go-homedir"

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

type removeContextTestCase struct {
	name string

	fakeConfig     *latest.Config
	fakeKubeConfig clientcmd.ClientConfig

	args             []string
	answers          []string
	graphQLResponses []interface{}
	provider         string
	all              bool
	providerList     []*cloudlatest.Provider

	expectedErr string
}

func TestRunRemoveContext(t *testing.T) {
	claimAsJSON, _ := json.Marshal(token.ClaimSet{
		Expiration: time.Now().Add(time.Hour).Unix(),
	})
	validEncodedClaim := base64.URLEncoding.EncodeToString(claimAsJSON)
	for strings.HasSuffix(string(validEncodedClaim), "=") {
		validEncodedClaim = strings.TrimSuffix(validEncodedClaim, "=")
	}

	testCases := []removeContextTestCase{
		removeContextTestCase{
			name:     "Delete all one spaces",
			provider: "myProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name:  "myProvider",
					Key:   "someKey",
					Token: "." + validEncodedClaim + ".",
				},
			},
			all: true,
			graphQLResponses: []interface{}{
				struct {
					Spaces []interface{} `json:"space"`
				}{
					Spaces: []interface{}{
						struct {
							Owner   struct{} `json:"account"`
							Context struct {
								Cluster struct{} `json:"cluster"`
							} `json:"kube_context"`
						}{},
					},
				},
			},
			fakeKubeConfig: &customKubeConfig{},
		},
		removeContextTestCase{
			name:     "Delete one context successfully",
			provider: "myProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name:  "myProvider",
					Key:   "someKey",
					Token: "." + validEncodedClaim + ".",
				},
			},
			graphQLResponses: []interface{}{
				struct {
					Spaces []interface{} `json:"space"`
				}{
					Spaces: []interface{}{
						struct {
							Owner   struct{} `json:"account"`
							Context struct {
								Cluster struct{} `json:"cluster"`
							} `json:"kube_context"`
						}{},
					},
				},
			},
			args: []string{"myContext"},
			fakeKubeConfig: &customKubeConfig{
				rawconfig: clientcmdapi.Config{
					Contexts: map[string]*clientcmdapi.Context{
						"myContext": &clientcmdapi.Context{},
					},
				},
			},
		},
		removeContextTestCase{
			name:     "Delete current context successfully",
			provider: "myProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name:  "myProvider",
					Key:   "someKey",
					Token: "." + validEncodedClaim + ".",
				},
			},
			graphQLResponses: []interface{}{
				struct {
					Spaces []interface{} `json:"space"`
				}{
					Spaces: []interface{}{
						struct {
							Owner   struct{} `json:"account"`
							Context struct {
								Cluster struct{} `json:"cluster"`
							} `json:"kube_context"`
						}{},
					},
				},
			},
			answers: []string{"current"},
			fakeKubeConfig: &customKubeConfig{
				rawconfig: clientcmdapi.Config{
					CurrentContext: "current",
					Contexts: map[string]*clientcmdapi.Context{
						"current": &clientcmdapi.Context{},
						"next":    &clientcmdapi.Context{},
					},
				},
			},
		},
	}

	log.SetInstance(&log.DiscardLogger{PanicOnExit: true})

	for _, testCase := range testCases {
		testRunRemoveContext(t, testCase)
	}
}

func testRunRemoveContext(t *testing.T, testCase removeContextTestCase) {
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

	for _, answer := range testCase.answers {
		survey.SetNextAnswer(answer)
	}

	cloudpkg.DefaultGraphqlClient = &customGraphqlClient{
		responses: testCase.graphQLResponses,
	}

	kubeconfig.SetFakeConfig(testCase.fakeKubeConfig)

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

	err = (&contextCmd{
		Provider:  testCase.provider,
		AllSpaces: testCase.all,
	}).RunRemoveContext(nil, testCase.args)

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
}*/
