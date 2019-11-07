package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	cloudlatest "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/mgutz/ansi"
	"k8s.io/client-go/tools/clientcmd"

	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type initTestCase struct {
	name string

	fakeConfig       *latest.Config
	fakeKubeConfig   clientcmd.ClientConfig
	fakeDockerClient docker.ClientInterface
	files            map[string]interface{}
	graphQLResponses []interface{}
	providerList     []*cloudlatest.Provider
	answers          []string

	reconfigureFlag bool
	dockerfileFlag  string
	contextFlag     string

	expectedOutput string
	expectedErr    string
	expectedConfig *latest.Config
}

func TestInit(t *testing.T) {
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
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
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

	_, readDirErr := ioutil.ReadFile(".")
	readDirError := strings.ReplaceAll(readDirErr.Error(), ".", "%s")

	testCases := []initTestCase{
		initTestCase{
			name: "Don't reconfigure the existing config",
			files: map[string]interface{}{
				constants.DefaultConfigPath: latest.Config{
					Version: latest.Version,
				},
			},
			expectedOutput: fmt.Sprintf("\nInfo Config already exists. If you want to recreate the config please run `devspace init --reconfigure`\nInfo \r         \nIf you want to continue with the existing config, run:\n- `%s` to develop application\n- `%s` to deploy application\n", ansi.Color("devspace dev", "white+b"), ansi.Color("devspace deploy", "white+b")),
			expectedConfig: &latest.Config{
				Version: latest.Version,
				Dev:     &latest.DevConfig{},
			},
		},
		initTestCase{
			name:           "Init with helm chart",
			answers:        []string{enterHelmChartOption, "someChart"},
			expectedOutput: fmt.Sprintf("\nDone Project successfully initialized\nInfo \r         \nPlease run: \n- `%s` to tell DevSpace to deploy to this namespace \n- `%s` to create a new space in DevSpace Cloud\n- `%s` to use an existing space\n", ansi.Color("devspace use namespace [NAME]", "white+b"), ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b")),
			expectedConfig: &latest.Config{
				Version: latest.Version,
				Deployments: []*latest.DeploymentConfig{
					&latest.DeploymentConfig{
						Name: filepath.Base(dir),
						Helm: &latest.HelmConfig{
							Chart: &latest.ChartConfig{
								Name: "someChart",
							},
						},
					},
				},
				Dev: &latest.DevConfig{},
			},
		},
		initTestCase{
			name: "Init with manifests",
			files: map[string]interface{}{
				filepath.Join(gitIgnoreFile, "someFile"): "",
			},
			answers:        []string{enterManifestsOption, "myManifest"},
			expectedOutput: fmt.Sprintf("\nWarn Error reading file .gitignore: "+readDirError+"\nDone Project successfully initialized\nInfo \r         \nPlease run: \n- `%s` to tell DevSpace to deploy to this namespace \n- `%s` to create a new space in DevSpace Cloud\n- `%s` to use an existing space\n", ".gitignore", ansi.Color("devspace use namespace [NAME]", "white+b"), ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b")),
			expectedConfig: &latest.Config{
				Version: latest.Version,
				Deployments: []*latest.DeploymentConfig{
					&latest.DeploymentConfig{
						Name: filepath.Base(dir),
						Kubectl: &latest.KubectlConfig{
							Manifests: []string{"myManifest"},
						},
					},
				},
				Dev: &latest.DevConfig{},
			},
		},
		initTestCase{
			name: "Init with existing image",
			files: map[string]interface{}{
				gitIgnoreFile: "",
			},
			answers:        []string{useExistingImageOption, "someImage", "1000", "1234"},
			expectedOutput: fmt.Sprintf("\nWarn Your application listens on a system port [0-1024]. Choose a forwarding-port to access your application via localhost.\nDone Project successfully initialized\nInfo \r         \nPlease run: \n- `%s` to tell DevSpace to deploy to this namespace \n- `%s` to create a new space in DevSpace Cloud\n- `%s` to use an existing space\n", ansi.Color("devspace use namespace [NAME]", "white+b"), ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b")),
			expectedConfig: &latest.Config{
				Version: latest.Version,
				Images: map[string]*latest.ImageConfig{
					"default": &latest.ImageConfig{
						Image:            "someImage",
						Tag:              "latest",
						CreatePullSecret: ptr.Bool(true),
						Build: &latest.BuildConfig{
							Disabled: ptr.Bool(true),
						},
					},
				},
				Deployments: []*latest.DeploymentConfig{
					&latest.DeploymentConfig{
						Name: filepath.Base(dir),
						Helm: &latest.HelmConfig{
							ComponentChart: ptr.Bool(true),
							Values: map[interface{}]interface{}{
								"containers": []*latest.ContainerConfig{
									{
										Image: "someImage",
									},
								},
								"service": &latest.ServiceConfig{
									Ports: []*latest.ServicePortConfig{
										{
											Port: ptr.Int(1000),
										},
									},
								},
							},
						},
					},
				},
				Dev: &latest.DevConfig{
					Ports: []*latest.PortForwardingConfig{
						&latest.PortForwardingConfig{
							ImageName: "default",
							PortMappings: []*latest.PortMapping{
								&latest.PortMapping{
									LocalPort:  ptr.Int(1234),
									RemotePort: ptr.Int(1000),
								},
							},
						},
					},
					Open: []*latest.OpenConfig{
						&latest.OpenConfig{
							URL: "http://localhost:1234",
						},
					},
				},
			},
		},
		initTestCase{
			name: "Entered existing Dockerfile",
			files: map[string]interface{}{
				"aDockerfile": "",
			},
			fakeDockerClient: &docker.FakeClient{
				AuthConfig: &dockertypes.AuthConfig{
					Username: "user",
					Password: "pass",
				},
			},
			answers: []string{enterDockerfileOption, "aDockerfile", "Use hub.docker.com => you are logged in as user", "user", "pass", "myImage", ""},
			expectedConfig: &latest.Config{
				Version: latest.Version,
				Images: map[string]*latest.ImageConfig{
					"default": &latest.ImageConfig{
						Image:      "",
						Dockerfile: "aDockerfile",
					},
				},
				Deployments: []*latest.DeploymentConfig{
					&latest.DeploymentConfig{
						Name: filepath.Base(dir),
						Helm: &latest.HelmConfig{
							ComponentChart: ptr.Bool(true),
							Values: map[interface{}]interface{}{
								"containers": []interface{}{
									struct{}{},
								},
							},
						},
					},
				},
				Dev: &latest.DevConfig{
					Sync: []*latest.SyncConfig{
						&latest.SyncConfig{
							ImageName:    "default",
							ExcludePaths: []string{"devspace.yaml"},
						},
					},
				},
			},
			expectedOutput: fmt.Sprintf("\nWait Checking registry authentication\nWarn You are not logged in to Docker Hub\nWarn Please make sure you have a https://hub.docker.com account\nWarn Installing docker is NOT required. You simply need a Docker Hub account\n\nWait Checking Docker credentials\nDone Project successfully initialized\nInfo \r         \nPlease run: \n- `%s` to tell DevSpace to deploy to this namespace \n- `%s` to create a new space in DevSpace Cloud\n- `%s` to use an existing space\n", ansi.Color("devspace use namespace [NAME]", "white+b"), ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b")),
		},
	}

	log.OverrideRuntimeErrorHandler(true)
	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testInit(t, testCase)
	}
}

func testInit(t *testing.T, testCase initTestCase) {
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

	for _, answer := range testCase.answers {
		survey.SetNextAnswer(answer)
	}

	providerConfig, err := cloudconfig.ParseProviderConfig()
	assert.NilError(t, err, "Error getting provider config in testCase %s", testCase.name)
	providerConfig.Providers = testCase.providerList

	configutil.SetFakeConfig(testCase.fakeConfig)
	configutil.ResetConfig()
	generated.ResetConfig()
	kubeconfig.SetFakeConfig(testCase.fakeKubeConfig)
	docker.SetFakeClient(testCase.fakeDockerClient)

	for path, content := range testCase.files {
		asYAML, err := yaml.Marshal(content)
		assert.NilError(t, err, "Error parsing config to yaml in testCase %s", testCase.name)
		err = fsutil.WriteToFile(asYAML, path)
		assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
	}

	err = (&InitCmd{
		Reconfigure: testCase.reconfigureFlag,
		Dockerfile:  testCase.dockerfileFlag,
		Context:     testCase.contextFlag,
	}).Run(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)

		config, err := configutil.GetConfig(nil)
		assert.NilError(t, err, "Error getting config after init call in testCase %s.", testCase.name)
		configYaml, err := yaml.Marshal(config)
		assert.NilError(t, err, "Error parsing config to yaml after init call in testCase %s.", testCase.name)
		expectedConfigYaml, err := yaml.Marshal(testCase.expectedConfig)
		assert.NilError(t, err, "Error parsing expected config to yaml after init call in testCase %s.", testCase.name)
		assert.Equal(t, string(configYaml), string(expectedConfigYaml), "Initialized config is wrong in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}

	err = filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
		os.RemoveAll(path)
		return nil
	})
	assert.NilError(t, err, "Error cleaning up in testCase %s", testCase.name)
}
