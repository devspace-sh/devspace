package kubectl

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/remotecache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	fakekube "github.com/loft-sh/devspace/pkg/devspace/kubectl/testing"
	log "github.com/loft-sh/devspace/pkg/util/log/testing"
	"github.com/loft-sh/devspace/pkg/util/ptr"

	yaml "gopkg.in/yaml.v3"
	"gotest.tools/assert"
)

type newTestCase struct {
	name         string
	kubeClient   kubectl.Client
	deployConfig *latest.DeploymentConfig

	expectedDeployer interface{}
	expectedErr      string
}

func TestNew(t *testing.T) {
	testCases := []newTestCase{
		{
			name: "No kubeClient",
			deployConfig: &latest.DeploymentConfig{
				Name: "someDeploy",
				Kubectl: &latest.KubectlConfig{
					KubectlBinaryPath: "someCmdPath",
					Manifests:         []string{"*someManifestkustomization.yaml"},
					Kustomize:         ptr.Bool(true),
				},
			},
			expectedDeployer: &DeployConfig{
				Name:      "someDeploy",
				CmdPath:   "someCmdPath",
				Manifests: []string{"someManifest"},

				DeploymentConfig: &latest.DeploymentConfig{
					Name: "someDeploy",
					Kubectl: &latest.KubectlConfig{
						KubectlBinaryPath: "someCmdPath",
						Manifests:         []string{"*someManifestkustomization.yaml"},
						Kustomize:         ptr.Bool(true),
					},
				},
			},
		},
		{
			name: "Everything given",
			deployConfig: &latest.DeploymentConfig{
				Name:      "someDeploy2",
				Namespace: "overwriteNamespace",
				Kubectl: &latest.KubectlConfig{
					KubectlBinaryPath: "someCmdPath2",
					Manifests:         []string{},
				},
			},
			kubeClient: &fakekube.Client{
				Context: "testContext",
			},
			expectedDeployer: &DeployConfig{
				Name:      "someDeploy2",
				CmdPath:   "someCmdPath2",
				Context:   "testContext",
				Namespace: "overwriteNamespace",
				DeploymentConfig: &latest.DeploymentConfig{
					Name:      "someDeploy2",
					Namespace: "overwriteNamespace",
					Kubectl: &latest.KubectlConfig{
						KubectlBinaryPath: "someCmdPath2",
						Manifests:         []string{},
					},
				},
			},
		},
		{
			name: "Inline Manifest",
			deployConfig: &latest.DeploymentConfig{
				Name: "someDeploy3",
				Kubectl: &latest.KubectlConfig{
					KubectlBinaryPath: "someCmdPath3",
					InlineManifest:    "inline: manifest",
				},
			},
			expectedDeployer: &DeployConfig{
				Name:           "someDeploy3",
				CmdPath:        "someCmdPath3",
				InlineManifest: "inline: manifest",

				DeploymentConfig: &latest.DeploymentConfig{
					Name: "someDeploy3",
					Kubectl: &latest.KubectlConfig{
						KubectlBinaryPath: "someCmdPath3",
						InlineManifest:    "inline: manifest",
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		if testCase.deployConfig == nil {
			testCase.deployConfig = &latest.DeploymentConfig{}
		}

		devCtx := devspacecontext.NewContext(context.Background(), nil, log.NewFakeLogger()).WithKubeClient(testCase.kubeClient)
		deployer, err := New(devCtx, testCase.deployConfig)
		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		deployerAsYaml, err := yaml.Marshal(deployer)
		assert.NilError(t, err, "Error marshaling deployer in testCase %s", testCase.name)
		expectationAsYaml, err := yaml.Marshal(testCase.expectedDeployer)
		fmt.Println(string(deployerAsYaml))
		fmt.Println("=========================================")
		fmt.Println(string(expectationAsYaml))
		assert.NilError(t, err, "Error marshaling expected deployer in testCase %s", testCase.name)
		assert.Equal(t, string(deployerAsYaml), string(expectationAsYaml), "Unexpected deployer in testCase %s", testCase.name)
	}
}

type renderTestCase struct {
	name string

	// output      string
	manifests []string
	kustomize bool
	// cache       *localcache.LocalCache
	// builtImages map[string]string

	expectedStreamOutput string
	expectedErr          string
}

// TODO: only for lint purpose, remove once the below test is fixed
var _ = renderTestCase{
	name:                 "",
	manifests:            []string{},
	kustomize:            false,
	expectedStreamOutput: "",
}

func TestRender(t *testing.T) {
	t.Skip("TODO: error:  no such file or directory")
	testCases := []renderTestCase{
		{
			name:                 "render one empty manifest",
			manifests:            []string{"mymanifest"},
			expectedStreamOutput: "\n---\n",
		},
	}

	for _, testCase := range testCases {
		cache := localcache.New(constants.DefaultCacheFolder)

		deployer := &DeployConfig{
			Manifests: testCase.manifests,
			DeploymentConfig: &latest.DeploymentConfig{
				Kubectl: &latest.KubectlConfig{
					Kustomize: &testCase.kustomize,
				},
			},
			CmdPath: "kubectl",
		}

		conf := config.NewConfig(map[string]interface{}{},
			map[string]interface{}{},
			&latest.Config{},
			cache,
			&remotecache.RemoteCache{},
			map[string]interface{}{},
			constants.DefaultConfigPath)

		devCtx := devspacecontext.NewContext(context.Background(), nil, log.NewFakeLogger()).WithConfig(conf)

		reader, writer := io.Pipe()
		defer reader.Close()

		go func() {
			defer writer.Close()

			err := deployer.Render(devCtx, writer)
			fmt.Println(err)
			if testCase.expectedErr == "" {
				assert.NilError(t, err, "Error in testCase %s", testCase.name)
			} else {
				assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
			}
		}()

		streamOutput, err := io.ReadAll(reader)
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
		{
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

		devCtx := devspacecontext.NewContext(context.Background(), nil, log.NewFakeLogger())

		status, err := deployer.Status(devCtx)

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
	name      string
	cmdPath   string
	manifests []string
	kustomize bool
	cache     *remotecache.RemoteCache
	// expectedDeployments []remotecache.DeploymentCache
	expectedErr   string
	expectedPaths []string
	expectedArgs  [][]string
}

func TestDelete(t *testing.T) {
	testCases := []deleteTestCase{
		{
			name:          "delete with one manifest",
			manifests:     []string{"oneManifest"},
			cmdPath:       "myPath",
			expectedPaths: []string{"myPath", "myPath"},
			expectedArgs: [][]string{
				{"create", "--dry-run", "--output", "yaml", "--validate=false", "--filename", "oneManifest"},
				{"delete", "--ignore-not-found=true", "-f", "-"},
			},
		},
	}

	for _, testCase := range testCases {
		cache := localcache.New("")
		deployer := &DeployConfig{
			CmdPath:   testCase.cmdPath,
			Manifests: testCase.manifests,
			DeploymentConfig: &latest.DeploymentConfig{
				Name: "someDeploy",
				Kubectl: &latest.KubectlConfig{
					Kustomize: &testCase.kustomize,
				},
			},
		}

		if testCase.cache == nil {
			testCase.cache = &remotecache.RemoteCache{}
		}

		conf := config.NewConfig(map[string]interface{}{},
			map[string]interface{}{},
			&latest.Config{},
			cache,
			&remotecache.RemoteCache{},
			map[string]interface{}{},
			constants.DefaultConfigPath)

		devCtx := devspacecontext.NewContext(context.Background(), nil, log.NewFakeLogger()).WithConfig(conf)

		err := Delete(devCtx, deployer.DeploymentConfig.Name)
		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		// statusAsYaml, err := yaml.Marshal(testCase.cache.Deployments)
		// assert.NilError(t, err, "Error marshaling status in testCase %s", testCase.name)
		// expectedAsYaml, err := yaml.Marshal(testCase.expectedDeployments)
		// assert.NilError(t, err, "Error marshaling expected status in testCase %s", testCase.name)
		// assert.Equal(t, string(statusAsYaml), string(expectedAsYaml), "Unexpected status in testCase %s", testCase.name)
	}
}

type deployTestCase struct {
	name string

	// output       string
	cmdPath      string
	context      string
	namespace    string
	manifests    []string
	kustomize    bool
	kubectlFlags []string
	cache        *remotecache.RemoteCache
	// forceDeploy  bool
	// builtImages  map[string]string

	expectedDeployed bool
	expectedErr      string
	expectedPaths    []string
	expectedArgs     [][]string
}

// TODO: only for lint purpose, remove once the below test is fixed
var _ = deployTestCase{
	name:             "",
	cmdPath:          "",
	context:          "",
	namespace:        "",
	manifests:        []string{},
	kustomize:        false,
	kubectlFlags:     []string{},
	cache:            &remotecache.RemoteCache{},
	expectedDeployed: false,
	expectedErr:      "",
	expectedPaths:    []string{},
	expectedArgs:     [][]string{},
}

func TestDeploy(t *testing.T) {
	t.Skip("TODO: executable file not found in $PATH")
	testCases := []deployTestCase{
		{
			name:             "deploy one manifest",
			cmdPath:          "myPath",
			context:          "myContext",
			namespace:        "myNamespace",
			manifests:        []string{"."},
			kubectlFlags:     []string{"someFlag"},
			expectedDeployed: true,
			expectedPaths:    []string{"myPath", "myPath"},
			expectedArgs: [][]string{
				{"create", "--context", "myContext", "--namespace", "myNamespace", "--dry-run", "--output", "yaml", "--validate=false", "--filename", "."},
				{"--context", "myContext", "--namespace", "myNamespace", "apply", "--force", "-f", "-", "someFlag"},
			},
		},
	}

	for _, testCase := range testCases {
		cache := localcache.New("")
		deployer := &DeployConfig{
			CmdPath:   testCase.cmdPath,
			Context:   testCase.context,
			Namespace: testCase.namespace,
			Manifests: testCase.manifests,
			DeploymentConfig: &latest.DeploymentConfig{
				Kubectl: &latest.KubectlConfig{
					Kustomize: &testCase.kustomize,
					ApplyArgs: testCase.kubectlFlags,
				},
			},
		}

		if testCase.cache == nil {
			testCase.cache = &remotecache.RemoteCache{}
		}

		conf := config.NewConfig(map[string]interface{}{},
			map[string]interface{}{},
			&latest.Config{},
			cache,
			&remotecache.RemoteCache{},
			map[string]interface{}{},
			constants.DefaultConfigPath)

		devCtx := devspacecontext.NewContext(context.Background(), nil, log.NewFakeLogger()).WithConfig(conf)

		deployed, err := deployer.Deploy(devCtx, false)

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
	cache        *localcache.LocalCache
	imageConfigs map[string]*latest.Image
	builtImages  map[string]string

	expectedRedeploy bool
	expectedManifest string
	expectedErr      string
}

// TODO: only for lint purpose, remove once the below test is fixed
var _ getReplacedManifestTestCase = getReplacedManifestTestCase{
	name:             "",
	cmdOutput:        nil,
	manifest:         "",
	kustomize:        false,
	cache:            &localcache.LocalCache{},
	imageConfigs:     map[string]*latest.Image{},
	builtImages:      map[string]string{},
	expectedRedeploy: false,
	expectedManifest: "",
	expectedErr:      "",
}

func TestGetReplacedManifest(t *testing.T) {
	t.Skip("TODO: manifest issue")
	testCases := []getReplacedManifestTestCase{
		{
			name:      "All empty",
			cmdOutput: "",
		},
		{
			name: "one replaced resource",
			cmdOutput: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Pod",
				"image":      "myimage",
			},
			cache: &localcache.LocalCache{
				Images: map[string]localcache.ImageCache{
					"myimage": {
						ImageName: "myimage",
						Tag:       "mytag",
					},
				},
			},
			imageConfigs: map[string]*latest.Image{
				"myimage": {
					Image: "myimage",
				},
			},
			builtImages: map[string]string{
				"myimage": "",
			},
			expectedRedeploy: true,
			expectedManifest: "apiVersion: v1\nimage: myimage:mytag\nkind: Pod\n",
		},
	}

	for _, testCase := range testCases {
		cache := localcache.New("")
		deployer := &DeployConfig{
			DeploymentConfig: &latest.DeploymentConfig{
				Kubectl: &latest.KubectlConfig{
					Kustomize: &testCase.kustomize,
				},
			},
		}

		conf := config.NewConfig(map[string]interface{}{},
			map[string]interface{}{},
			&latest.Config{
				Images: testCase.imageConfigs,
			},
			cache,
			&remotecache.RemoteCache{},
			map[string]interface{}{},
			constants.DefaultConfigPath)

		devCtx := devspacecontext.NewContext(context.Background(), nil, log.NewFakeLogger()).WithConfig(conf)

		shouldRedeploy, replacedManifest, _, err := deployer.getReplacedManifest(devCtx, false, testCase.manifest)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		assert.Equal(t, shouldRedeploy, testCase.expectedRedeploy, "Unexpected shouldRedeploy-bool in testCase %s", testCase.name)
		assert.Equal(t, replacedManifest, testCase.expectedManifest, "Unexpected replaced manifest in testCase %s", testCase.name)
	}
}
