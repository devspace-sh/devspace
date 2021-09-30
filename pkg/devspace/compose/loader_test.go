package compose

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
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

	assert.DeepEqual(t, toDeploymentMap(expectedConfig.Deployments), toDeploymentMap(actualConfig.Deployments))

	actualConfig.Deployments = nil
	expectedConfig.Deployments = nil
	assert.DeepEqual(t, expectedConfig, actualConfig)
}

func toDeploymentMap(deployments []*latest.DeploymentConfig) map[string]latest.DeploymentConfig {
	deploymentMap := map[string]latest.DeploymentConfig{}
	for _, deployment := range deployments {
		deploymentMap[deployment.Name] = *deployment
	}
	return deploymentMap
}
