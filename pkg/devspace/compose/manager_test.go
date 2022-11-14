package compose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	"gopkg.in/yaml.v3"
	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
)

func TestLoad(t *testing.T) {
	dirs, err := os.ReadDir("testdata")
	if err != nil {
		t.Error(err)
	}

	if len(dirs) == 0 {
		t.Error("No test cases found. Add some!")
	}

	focused := []string{}
	for _, dir := range dirs {
		if strings.HasPrefix(dir.Name(), "f_") {
			focused = append(focused, dir.Name())
		}
	}

	if len(focused) > 0 {
		for _, focus := range focused {
			testLoad(focus, t)
		}
	} else {
		for _, dir := range dirs {
			if !strings.HasPrefix(dir.Name(), "x_") {
				testLoad(dir.Name(), t)
			}
		}
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
	dockerCompose, err := LoadDockerComposeProject(dockerComposePath)
	if err != nil {
		t.Errorf("Unexpected error occurred loading the docker-compose.yaml: %s", err.Error())
	}
	loader := NewComposeManager(dockerCompose)

	actualError := loader.Load(log.Discard)

	if actualError != nil {
		expectedError, err := os.ReadFile("error.txt")
		if err != nil {
			t.Errorf("Unexpected error occurred loading the docker-compose.yaml: %s", err.Error())
		}

		assert.Equal(t, string(expectedError), actualError.Error(), "Expected error:\n%s\nbut got:\n%s\n in testCase %s", string(expectedError), actualError.Error(), dir)
	}

	for path, actualConfig := range loader.Configs() {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("Please create the expected DevSpace configuration by creating a %s in the testdata/%s folder", path, dir)
		}

		expectedConfig := &latest.Config{}
		err = yaml.Unmarshal(data, expectedConfig)
		if err != nil {
			t.Errorf("Error unmarshaling the expected configuration: %s", err.Error())
		}

		assert.Check(
			t,
			cmp.DeepEqual(expectedConfig, actualConfig),
			"configs did not match in test case %s",
			dir,
		)
	}

}
