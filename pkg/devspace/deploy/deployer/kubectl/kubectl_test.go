package kubectl

import (
	"io"
	"io/ioutil"
	"os/exec"
	"strings"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/deployer"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	fakekube "github.com/devspace-cloud/devspace/pkg/devspace/kubectl/testing"
	"github.com/devspace-cloud/devspace/pkg/util/command"
	log "github.com/devspace-cloud/devspace/pkg/util/log/testing"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type newTestCase struct {
	name string

	config       *latest.Config
	kubeClient   kubectl.Client
	deployConfig *latest.DeploymentConfig

	expectedDeployer interface{}
	expectedErr      string
}

func TestNew(t *testing.T) {
	testCases := []newTestCase{
		newTestCase{
			name:        "No kubectl",
			expectedErr: "Error creating kubectl deploy config: kubectl is nil",
		},
		newTestCase{
			name: "No manifests",
			deployConfig: &latest.DeploymentConfig{
				Kubectl: &latest.KubectlConfig{},
			},
			expectedErr: "No manifests defined for kubectl deploy",
		},
		newTestCase{
			name: "No kubeClient",
			deployConfig: &latest.DeploymentConfig{
				Name: "someDeploy",
				Kubectl: &latest.KubectlConfig{
					CmdPath:   "someCmdPath",
					Manifests: []string{"*someManifestkustomization.yaml"},
					Kustomize: ptr.Bool(true),
				},
			},
			expectedDeployer: &DeployConfig{
				Name:      "someDeploy",
				CmdPath:   "someCmdPath",
				Manifests: []string{"someManifest"},

				DeploymentConfig: &latest.DeploymentConfig{
					Name: "someDeploy",
					Kubectl: &latest.KubectlConfig{
						CmdPath:   "someCmdPath",
						Manifests: []string{"*someManifestkustomization.yaml"},
						Kustomize: ptr.Bool(true),
					},
				},
			},
		},
		newTestCase{
			name: "Everything given",
			deployConfig: &latest.DeploymentConfig{
				Name:      "someDeploy2",
				Namespace: "overwriteNamespace",
				Kubectl: &latest.KubectlConfig{
					CmdPath:   "someCmdPath2",
					Manifests: []string{},
				},
			},
			kubeClient: &fakekube.Client{},
			expectedDeployer: &DeployConfig{
				Name:       "someDeploy2",
				KubeClient: &fakekube.Client{},
				CmdPath:    "someCmdPath2",
				Context:    "testContext",
				Namespace:  "overwriteNamespace",

				DeploymentConfig: &latest.DeploymentConfig{
					Name:      "someDeploy2",
					Namespace: "overwriteNamespace",
					Kubectl: &latest.KubectlConfig{
						CmdPath:   "someCmdPath2",
						Manifests: []string{},
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		if testCase.deployConfig == nil {
			testCase.deployConfig = &latest.DeploymentConfig{}
		}

		deployer, err := New(testCase.config, testCase.kubeClient, testCase.deployConfig, nil)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		deployerAsYaml, err := yaml.Marshal(deployer)
		assert.NilError(t, err, "Error marshaling deployer in testCase %s", testCase.name)
		expectationAsYaml, err := yaml.Marshal(testCase.expectedDeployer)
		assert.NilError(t, err, "Error marshaling expected deployer in testCase %s", testCase.name)
		assert.Equal(t, string(deployerAsYaml), string(expectationAsYaml), "Unexpected deployer in testCase %s", testCase.name)
	}
}

type fakeExecuter struct {
	output interface{}
	err    error

	t            *testing.T
	expectedPath []string
	expectedArgs [][]string
	testCase     string
}

func (e *fakeExecuter) RunCommand(path string, args []string) ([]byte, error) {
	e.checkParams(path, args)

	if output, ok := e.output.(string); ok {
		return []byte(output), e.err
	}

	yamlOutput, err := yaml.Marshal(e.output)
	if err != nil {
		return nil, errors.Wrap(err, "marshal output")
	}
	return yamlOutput, e.err
}

func (e *fakeExecuter) GetCommand(path string, args []string) command.Interface {
	e.checkParams(path, args)
	return &command.FakeCommand{}
}

func (e *fakeExecuter) checkParams(path string, args []string) {
	if e.t != nil {
		assert.Equal(e.t, path, e.expectedPath[0], "Unexpected path in testCase %s", e.testCase)
		assert.Equal(e.t, strings.Join(args, ","), strings.Join(e.expectedArgs[0], ","), "Unexpected args in testCase %s", e.testCase)

		e.expectedPath = e.expectedPath[1:]
		e.expectedArgs = e.expectedArgs[1:]
	}
}

type renderTestCase struct {
	name string

	output      string
	manifests   []string
	kustomize   bool
	cache       *generated.CacheConfig
	builtImages map[string]string

	expectedStreamOutput string
	expectedErr          string
}

func TestRender(t *testing.T) {
	testCases := []renderTestCase{
		renderTestCase{
			name:                 "render one empty manifest",
			manifests:            []string{"mymanifest"},
			expectedStreamOutput: "\n---\n",
		},
	}

	for _, testCase := range testCases {
		deployer := &DeployConfig{
			Manifests: testCase.manifests,
			DeploymentConfig: &latest.DeploymentConfig{
				Kubectl: &latest.KubectlConfig{
					Kustomize: &testCase.kustomize,
				},
			},
			commandExecuter: &fakeExecuter{
				output: testCase.output,
			},
		}

		reader, writer := io.Pipe()
		defer reader.Close()

		go func() {
			defer writer.Close()

			err := deployer.Render(testCase.cache, testCase.builtImages, writer)

			if testCase.expectedErr == "" {
				assert.NilError(t, err, "Error in testCase %s", testCase.name)
			} else {
				assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
			}
		}()

		streamOutput, err := ioutil.ReadAll(reader)
		assert.NilError(t, err, "Error reading stream in testCase %s", testCase.name)
		assert.Equal(t, string(streamOutput), testCase.expectedStreamOutput, "Unexpected stream output in testCase %s", testCase.name)
	}
}

type statusTestCase struct {
	name string

	deployername string
	manifests    []string

	expectedStatus deployer.StatusResult
	expectedErr    string
}

func TestStatus(t *testing.T) {
	testCases := []statusTestCase{
		statusTestCase{
			name:         "Too long manifests",
			deployername: "longManifestDeployer",
			manifests:    []string{"ThatIsAVeryLongManifestNameHereJustTooLargeToFitOnAConsole"},
			expectedStatus: deployer.StatusResult{
				Name:   "longManifestDeployer",
				Type:   "Manifests",
				Target: "ThatIsAVeryLongManif...",
				Status: "N/A",
			},
		},
	}

	for _, testCase := range testCases {
		deployer := &DeployConfig{
			Name:      testCase.deployername,
			Manifests: testCase.manifests,
		}

		status, err := deployer.Status()

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		statusAsYaml, err := yaml.Marshal(status)
		assert.NilError(t, err, "Error marshaling status in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedStatus)
		assert.NilError(t, err, "Error marshaling expected status in testCase %s", testCase.name)
		assert.Equal(t, string(statusAsYaml), string(expectedAsYaml), "Unexpected status in testCase %s", testCase.name)
	}
}

type deleteTestCase struct {
	name string

	output    string
	cmdPath   string
	manifests []string
	kustomize bool
	cache     *generated.CacheConfig

	expectedDeployments map[string]*generated.DeploymentCache
	expectedErr         string
	expectedPaths       []string
	expectedArgs        [][]string
}

func TestDelete(t *testing.T) {
	testCases := []deleteTestCase{
		deleteTestCase{
			name:          "delete with one manifest",
			manifests:     []string{"oneManifest"},
			cmdPath:       "myPath",
			expectedPaths: []string{"myPath", "myPath"},
			expectedArgs: [][]string{
				[]string{"create", "--dry-run", "--output,yaml", "--validate=false", "--filename", "oneManifest"},
				[]string{"delete", "--ignore-not-found=true", "-f", "-"},
			},
		},
	}

	for _, testCase := range testCases {
		deployer := &DeployConfig{
			CmdPath:   testCase.cmdPath,
			Manifests: testCase.manifests,
			DeploymentConfig: &latest.DeploymentConfig{
				Name: "someDeploy",
				Kubectl: &latest.KubectlConfig{
					Kustomize: &testCase.kustomize,
				},
			},
			commandExecuter: &fakeExecuter{
				output:       testCase.output,
				t:            t,
				testCase:     testCase.name,
				expectedPath: testCase.expectedPaths,
				expectedArgs: testCase.expectedArgs,
			},
			Log: &log.FakeLogger{},
		}

		if testCase.cache == nil {
			testCase.cache = &generated.CacheConfig{
				Deployments: map[string]*generated.DeploymentCache{},
			}
		}

		err := deployer.Delete(testCase.cache)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		statusAsYaml, err := yaml.Marshal(testCase.cache.Deployments)
		assert.NilError(t, err, "Error marshaling status in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedDeployments)
		assert.NilError(t, err, "Error marshaling expected status in testCase %s", testCase.name)
		assert.Equal(t, string(statusAsYaml), string(expectedAsYaml), "Unexpected status in testCase %s", testCase.name)
	}
}

type deployTestCase struct {
	name string

	output       string
	cmdPath      string
	context      string
	namespace    string
	manifests    []string
	kustomize    bool
	kubectlFlags []string
	cache        *generated.CacheConfig
	forceDeploy  bool
	builtImages  map[string]string

	expectedDeployed bool
	expectedErr      string
	expectedPaths    []string
	expectedArgs     [][]string
}

func TestDeploy(t *testing.T) {
	testCases := []deployTestCase{
		deployTestCase{
			name:             "deploy one manifest",
			cmdPath:          "myPath",
			context:          "myContext",
			namespace:        "myNamespace",
			manifests:        []string{"/"},
			kubectlFlags:     []string{"someFlag"},
			expectedDeployed: true,
			expectedPaths:    []string{"myPath", "myPath"},
			expectedArgs: [][]string{
				[]string{"create", "--context", "myContext", "--namespace", "myNamespace", "--dry-run", "--output,yaml", "--validate=false", "--filename", "/"},
				[]string{"--context", "myContext", "--namespace", "myNamespace", "apply", "--force", "-f", "-", "someFlag"},
			},
		},
	}

	for _, testCase := range testCases {
		deployer := &DeployConfig{
			CmdPath:   testCase.cmdPath,
			Context:   testCase.context,
			Namespace: testCase.namespace,
			Manifests: testCase.manifests,
			DeploymentConfig: &latest.DeploymentConfig{
				Kubectl: &latest.KubectlConfig{
					Kustomize: &testCase.kustomize,
					Flags:     testCase.kubectlFlags,
				},
			},
			commandExecuter: &fakeExecuter{
				output:       testCase.output,
				t:            t,
				testCase:     testCase.name,
				expectedPath: testCase.expectedPaths,
				expectedArgs: testCase.expectedArgs,
			},
			Log: &log.FakeLogger{},
		}

		if testCase.cache == nil {
			testCase.cache = &generated.CacheConfig{
				Deployments: map[string]*generated.DeploymentCache{},
			}
		}

		deployed, err := deployer.Deploy(testCase.cache, testCase.forceDeploy, testCase.builtImages)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		assert.Equal(t, deployed, testCase.expectedDeployed, "Unexpected deployed-bool in testCase %s", testCase.name)
	}
}

type getReplacedManifestTestCase struct {
	name string

	cmdOutput    interface{}
	manifest     string
	kustomize    bool
	cache        *generated.CacheConfig
	imageConfigs map[string]*latest.ImageConfig
	builtImages  map[string]string

	expectedRedeploy bool
	expectedManifest string
	expectedErr      string
}

func TestGetReplacedManifest(t *testing.T) {
	testCases := []getReplacedManifestTestCase{
		getReplacedManifestTestCase{
			name:      "All empty",
			cmdOutput: "",
		},
		getReplacedManifestTestCase{
			name: "one replaced resource",
			cmdOutput: map[string]interface{}{
				"apiVersion": 1,
				"image":      "myimage",
			},
			cache: &generated.CacheConfig{
				Images: map[string]*generated.ImageCache{
					"myimage": &generated.ImageCache{
						ImageName: "myimage",
						Tag:       "mytag",
					},
				},
			},
			imageConfigs: map[string]*latest.ImageConfig{
				"myimage": &latest.ImageConfig{
					Image: "myimage",
				},
			},
			builtImages: map[string]string{
				"myimage": "",
			},
			expectedRedeploy: true,
			expectedManifest: "apiVersion: 1\nimage: myimage:mytag\n",
		},
	}

	for _, testCase := range testCases {
		deployer := &DeployConfig{
			DeploymentConfig: &latest.DeploymentConfig{
				Kubectl: &latest.KubectlConfig{
					Kustomize: &testCase.kustomize,
				},
			},
			commandExecuter: &fakeExecuter{
				output: testCase.cmdOutput,
			},
			config: &latest.Config{
				Images: testCase.imageConfigs,
			},
		}

		shouldRedeploy, replacedManifest, err := deployer.getReplacedManifest(testCase.manifest, testCase.cache, testCase.builtImages)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		assert.Equal(t, shouldRedeploy, testCase.expectedRedeploy, "Unexpected shouldRedeploy-bool in testCase %s", testCase.name)
		assert.Equal(t, replacedManifest, testCase.expectedManifest, "Unexpected replaced manifest in testCase %s", testCase.name)
	}
}

type dryRunTestCase struct {
	name string

	manifest  string
	context   string
	namespace string
	cmdPath   string
	kustomize bool

	cmdOutput string
	cmdErr    error

	expectedStreamOutput string
	expectedErr          string
	expectedPath         string
	expectedArgs         []string
}

func TestDryRun(t *testing.T) {
	testCases := []dryRunTestCase{
		dryRunTestCase{
			name:     "command returns error",
			cmdPath:  "path1",
			manifest: "manifest1",
			cmdErr: &exec.ExitError{
				Stderr: []byte("Test std err"),
			},
			expectedErr:  "Test std err",
			expectedPath: "path1",
			expectedArgs: []string{"create", "--dry-run", "--output", "yaml", "--validate=false", "--filename", "manifest1"},
		},
		dryRunTestCase{
			name:                 "all args, no error",
			cmdPath:              "path2",
			manifest:             "manifest2",
			context:              "mycontext",
			namespace:            "mynamespace",
			kustomize:            true,
			cmdOutput:            "cmdOutput",
			expectedStreamOutput: "cmdOutput",
			expectedPath:         "path2",
			expectedArgs:         []string{"create", "--context", "mycontext", "--namespace", "mynamespace", "--dry-run", "--output", "yaml", "--validate=false", "--kustomize", "manifest2"},
		},
	}

	for _, testCase := range testCases {
		deployer := &DeployConfig{
			Context:   testCase.context,
			Namespace: testCase.namespace,
			CmdPath:   testCase.cmdPath,
			DeploymentConfig: &latest.DeploymentConfig{
				Kubectl: &latest.KubectlConfig{
					Kustomize: &testCase.kustomize,
				},
			},
			commandExecuter: &fakeExecuter{
				output:       testCase.cmdOutput,
				err:          testCase.cmdErr,
				t:            t,
				expectedPath: []string{testCase.expectedPath},
				expectedArgs: [][]string{testCase.expectedArgs},
				testCase:     testCase.name,
			},
		}

		output, err := deployer.dryRun(testCase.manifest)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		assert.Equal(t, string(output), testCase.expectedStreamOutput, "Unexpected output in testCase %s", testCase.name)
	}
}
