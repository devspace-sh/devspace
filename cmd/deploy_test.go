package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devspace-cloud/devspace/cmd/flags"
	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	cloudlatest "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type deployTestCase struct {
	name string

	fakeConfig       *latest.Config
	fakeKubeConfig   clientcmd.ClientConfig
	fakeKubeClient   *kubectl.Client
	files            map[string]interface{}
	graphQLResponses []interface{}
	providerList     []*cloudlatest.Provider

	forceBuildFlag              bool
	skipBuildFlag               bool
	buildSequentialFlag         bool
	forceDeployFlag             bool
	deploymentsFlag             string
	forceDependenciesFlag       bool
	skipPushFlag                bool
	allowCyclicDependenciesFlag bool
	globalFlags                 flags.GlobalFlags

	expectedOutput string
	expectedErr    string
}

func TestDeploy(t *testing.T) {
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

	testCases := []deployTestCase{
		deployTestCase{
			name:           "Invalid flags",
			fakeConfig:     &latest.Config{},
			skipBuildFlag:  true,
			forceBuildFlag: true,
			expectedErr:    "Flags --skip-build & --force-build cannot be used together",
		},
		deployTestCase{
			name: "Cyclic dependency",
			fakeConfig: &latest.Config{
				Version: "v1beta3",
				Dependencies: []*latest.DependencyConfig{
					&latest.DependencyConfig{
						Source: &latest.SourceConfig{
							Path: "dependency1",
						},
					},
				},
			},
			fakeKubeClient: &kubectl.Client{
				Client:         fake.NewSimpleClientset(),
				CurrentContext: "minikube",
			},
			fakeKubeConfig: &customKubeConfig{
				rawconfig: clientcmdapi.Config{
					Contexts: map[string]*clientcmdapi.Context{
						"minikube": &clientcmdapi.Context{},
					},
					AuthInfos: map[string]*clientcmdapi.AuthInfo{
						"": &clientcmdapi.AuthInfo{},
					},
				},
			},
			files: map[string]interface{}{
				"devspace.yaml": &latest.Config{
					Version: "v1beta3",
					Dependencies: []*latest.DependencyConfig{
						&latest.DependencyConfig{
							Source: &latest.SourceConfig{
								Path: "dependency1",
							},
						},
					},
				},
				"dependency1/devspace.yaml": &latest.Config{
					Version: "v1beta3",
					Dependencies: []*latest.DependencyConfig{
						&latest.DependencyConfig{
							Source: &latest.SourceConfig{
								Path: "..",
							},
						},
					},
				},
			},
			expectedErr:    fmt.Sprintf("deploy dependencies: Cyclic dependency found: \n%s\n%s\n%s.\n To allow cyclic dependencies run with the '%s' flag", filepath.Join(dir, "dependency1"), dir, filepath.Join(dir, "dependency1"), ansi.Color("--allow-cyclic", "white+b")),
			expectedOutput: fmt.Sprintf("\nInfo Using kube context '%s'\nInfo Using namespace '%s'\nDone Created namespace: \nInfo Start resolving dependencies", ansi.Color("minikube", "white+b"), ansi.Color("", "white+b")),
		},
		deployTestCase{
			name:       "Successfully deployed nothing",
			fakeConfig: &latest.Config{},
			fakeKubeClient: &kubectl.Client{
				Client:         fake.NewSimpleClientset(),
				CurrentContext: "minikube",
			},
			fakeKubeConfig: &customKubeConfig{
				rawconfig: clientcmdapi.Config{
					Contexts: map[string]*clientcmdapi.Context{
						"minikube": &clientcmdapi.Context{},
					},
					AuthInfos: map[string]*clientcmdapi.AuthInfo{
						"": &clientcmdapi.AuthInfo{},
					},
				},
			},
			deploymentsFlag: " ",
			expectedOutput:  fmt.Sprintf("\nInfo Using kube context '%s'\nInfo Using namespace '%s'\nDone Created namespace: \nDone Successfully deployed!\nInfo \r         \nRun: \n- `%s` to create an ingress for the app and open it in the browser \n- `%s` to open a shell into the container \n- `%s` to show the container logs\n- `%s` to open the management ui\n- `%s` to analyze the space for potential issues\n", ansi.Color("minikube", "white+b"), ansi.Color("", "white+b"), ansi.Color("devspace open", "white+b"), ansi.Color("devspace enter", "white+b"), ansi.Color("devspace logs", "white+b"), ansi.Color("devspace ui", "white+b"), ansi.Color("devspace analyze", "white+b")),
		},
	}

	log.OverrideRuntimeErrorHandler(true)
	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testDeploy(t, testCase)
	}
}

func testDeploy(t *testing.T, testCase deployTestCase) {
	logOutput = ""

	defer func() {
		for path := range testCase.files {
			removeTask := strings.Split(path, "/")[0]
			err := os.RemoveAll(removeTask)
			assert.NilError(t, err, "Error cleaning up folder in testCase %s", testCase.name)
		}
		err := os.RemoveAll(log.Logdir)
		assert.NilError(t, err, "Error cleaning up folder in testCase %s", testCase.name)
	}()

	cloudpkg.DefaultGraphqlClient = &customGraphqlClient{
		responses: testCase.graphQLResponses,
	}

	providerConfig, err := cloudconfig.ParseProviderConfig()
	assert.NilError(t, err, "Error getting provider config in testCase %s", testCase.name)
	providerConfig.Providers = testCase.providerList

	configutil.SetFakeConfig(testCase.fakeConfig)
	generated.ResetConfig()
	kubeconfig.SetFakeConfig(testCase.fakeKubeConfig)
	kubectl.SetFakeClient(testCase.fakeKubeClient)

	for path, content := range testCase.files {
		asYAML, err := yaml.Marshal(content)
		assert.NilError(t, err, "Error parsing config to yaml in testCase %s", testCase.name)
		err = fsutil.WriteToFile(asYAML, path)
		assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
	}

	err = (&DeployCmd{
		GlobalFlags: &testCase.globalFlags,

		ForceBuild:        testCase.forceBuildFlag,
		SkipBuild:         testCase.skipBuildFlag,
		BuildSequential:   testCase.buildSequentialFlag,
		ForceDeploy:       testCase.forceDeployFlag,
		Deployments:       testCase.deploymentsFlag,
		ForceDependencies: testCase.forceDependenciesFlag,

		SkipPush:                testCase.skipPushFlag,
		AllowCyclicDependencies: testCase.allowCyclicDependenciesFlag,
	}).Run(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}

	err = filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
		os.RemoveAll(path)
		return nil
	})
	assert.NilError(t, err, "Error cleaning up in testCase %s", testCase.name)
}
