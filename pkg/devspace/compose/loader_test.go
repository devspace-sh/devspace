package compose

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	composeloader "github.com/compose-spec/compose-go/loader"
	composetypes "github.com/compose-spec/compose-go/types"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/hash"
	"github.com/loft-sh/devspace/pkg/util/log"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
)

func TestLoad(t *testing.T) {
	dirs, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Error(err)
	}

	if len(dirs) == 0 {
		t.Error("No test cases found. Add some!")
	}

	for _, dir := range dirs {
		testLoad(dir.Name(), t)
	}
}

func testLoad(dir string, t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}

	err = os.Chdir(filepath.Join(wd, "testdata", dir))
	if err != nil {
		t.Error(err)
	}
	defer func() {
		err := os.Chdir(wd)
		if err != nil {
			t.Error(err)
		}
	}()

	dockerComposePath := GetDockerComposePath()
	loader := NewDockerComposeLoader(dockerComposePath)

	actualConfig, actualError := loader.Load(log.Discard)
	if actualError != nil {
		expectedError, err := ioutil.ReadFile("error.txt")
		if err != nil {
			t.Errorf("Unexpected error occurred loading the docker-compose.yaml: %s", err.Error())
		}

		assert.Equal(t, string(expectedError), actualError.Error(), "Expected error:\n%s\nbut got:\n%s\n in testCase %s", string(expectedError), actualError.Error(), dir)
	}

	data, err := ioutil.ReadFile("expected.yaml")
	if err != nil {
		t.Errorf("Please create the expected DevSpace configuration by creating a expected.yaml in the testdata/%s folder", dir)
	}

	expectedConfig := &latest.Config{}
	err = yaml.Unmarshal(data, expectedConfig)
	if err != nil {
		t.Errorf("Error unmarshaling the expected configuration: %s", err.Error())
	}

	assert.Check(
		t,
		cmp.DeepEqual(toDeploymentMap(expectedConfig.Deployments), toDeploymentMap(actualConfig.Deployments)),
		"deployment properties did not match in test case %s",
		dir,
	)
	actualDeployments := actualConfig.Deployments
	actualConfig.Deployments = nil
	expectedConfig.Deployments = nil

	assert.Check(
		t,
		cmp.DeepEqual(toWaitHookMap(expectedConfig.Hooks), toWaitHookMap(actualConfig.Hooks)),
		"hook properties did not match in test case %s",
		dir,
	)
	actualHooks := actualConfig.Hooks
	actualConfig.Hooks = nil
	expectedConfig.Hooks = nil

	assert.Check(
		t,
		cmp.DeepEqual(expectedConfig, actualConfig),
		"config properties did not match in test case %s",
		dir,
	)

	// Load docker compose to determine dependency ordering
	content, err := ioutil.ReadFile(dockerComposePath)
	if err != nil {
		t.Errorf("Unexpected error occurred loading the docker-compose.yaml: %s", err.Error())
	}
	dockerCompose, err := composeloader.Load(composetypes.ConfigDetails{
		ConfigFiles: []composetypes.ConfigFile{
			{
				Content: content,
			},
		},
	})
	if err != nil {
		t.Error(err)
	}

	// Determine which deployments should have wait hooks
	expectedWaitHooks := map[string]bool{}
	err = dockerCompose.WithServices(nil, func(service composetypes.ServiceConfig) error {
		for _, dep := range service.GetDependencies() {
			expectedWaitHooks[dep] = true
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}

	// Iterate services in dependency order
	err = dockerCompose.WithServices(nil, func(service composetypes.ServiceConfig) error {
		waitHookIdx := getWaitHookIndex(service.Name, actualHooks)

		for _, dep := range service.GetDependencies() {
			// Check deployments order
			assert.Check(t, getDeploymentIndex(dep, actualDeployments) < getDeploymentIndex(service.Name, actualDeployments), "%s deployment should come after %s for test case %s", service.Name, dep, dir)

			// Check for wait hook order
			_, ok := expectedWaitHooks[service.Name]
			if ok {
				assert.Check(t, getWaitHookIndex(dep, actualHooks) < waitHookIdx, "%s wait hook should come after %s", service.Name, dep)
			}
		}

		uploadDoneHookIdx := getUploadDoneHookIndex(service.Name, actualHooks)
		if uploadDoneHookIdx != -1 {
			// Check that upload done hooks come before wait hooks
			if waitHookIdx != -1 {
				assert.Check(t, uploadDoneHookIdx < waitHookIdx, "%s wait hook should come after upload done hooks for test case %s", service.Name, dir)
			}

			// Check that upload hooks come before upload done hooks
			for idx, hook := range actualHooks {
				if hook.Upload != nil && hook.Container.ContainerName == UploadVolumesContainerName && hook.Container.LabelSelector != nil && hook.Container.LabelSelector["app.kubernetes.io/component"] == service.Name {
					assert.Check(t, idx < uploadDoneHookIdx, "%s upload done hook should come after upload hooks for test case %s", service.Name, dir)
				}
			}
		}

		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func toDeploymentMap(deployments []*latest.DeploymentConfig) map[string]latest.DeploymentConfig {
	deploymentMap := map[string]latest.DeploymentConfig{}
	for _, deployment := range deployments {
		deploymentMap[deployment.Name] = *deployment
	}
	return deploymentMap
}

func toWaitHookMap(hooks []*latest.HookConfig) map[string]latest.HookConfig {
	hookMap := map[string]latest.HookConfig{}
	for _, hook := range hooks {
		out, _ := yaml.Marshal(hook)
		hookKey := hash.String(string(out))
		hookMap[hookKey] = *hook
	}
	return hookMap
}

func getDeploymentIndex(name string, deployments []*latest.DeploymentConfig) int {
	for idx, deployment := range deployments {
		if deployment.Name == name {
			return idx
		}
	}
	return -1
}

func getWaitHookIndex(name string, hooks []*latest.HookConfig) int {
	for idx, hook := range hooks {
		if hook.Wait != nil && hook.Container != nil && hook.Container.LabelSelector != nil && hook.Container.LabelSelector["app.kubernetes.io/component"] == name {
			return idx
		}
	}
	return -1
}

func getUploadDoneHookIndex(name string, hooks []*latest.HookConfig) int {
	for idx, hook := range hooks {
		if hook.Command == "touch /tmp/done" && hook.Container != nil && hook.Container.LabelSelector != nil && hook.Container.LabelSelector["app.kubernetes.io/component"] == name {
			return idx
		}
	}
	return -1
}
