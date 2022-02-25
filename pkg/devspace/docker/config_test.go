package docker

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	configtypes "github.com/docker/cli/cli/config/types"
	"github.com/docker/docker/api/types"
	"github.com/loft-sh/devspace/pkg/util/fsutil"
	"gopkg.in/yaml.v3"
	"gotest.tools/assert"
)

type getAllAuthConfigsTestCase struct {
	name string

	files map[string]interface{}

	expectedAuthConfigs map[string]types.AuthConfig
	expectedErr         bool
}

func TestGetAllAuthConfigs(t *testing.T) {
	testCases := []getAllAuthConfigsTestCase{
		{
			name: "empty dir",
		},
		{
			name: "filled dir",
			files: map[string]interface{}{
				"config.json": configfile.ConfigFile{
					AuthConfigs: map[string]configtypes.AuthConfig{
						"key": {
							Username:      "ValUser",
							Password:      "ValPass",
							Email:         "ValEmail",
							ServerAddress: "ValServerAddress",
							IdentityToken: "ValIdentityToken",
							RegistryToken: "ValRegistryToken",
						},
					},
				},
			},
			expectedAuthConfigs: map[string]types.AuthConfig{
				"key": {
					Username:      "ValUser",
					Password:      "ValPass",
					Email:         "ValEmail",
					ServerAddress: "key",
					IdentityToken: "ValIdentityToken",
					RegistryToken: "ValRegistryToken",
				},
			},
		},
	}

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
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}()

	configDir = dir

	for _, testCase := range testCases {
		for path, content := range testCase.files {
			asJSON, err := json.Marshal(content)
			assert.NilError(t, err, "Error parsing content to json in testCase %s", testCase.name)
			if content == "" {
				asJSON = []byte{}
			}
			err = fsutil.WriteToFile(asJSON, path)
			assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
		}

		authconfigs, err := GetAllAuthConfigs()

		if !testCase.expectedErr {
			assert.NilError(t, err, "Unexpected error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected error %v in testCase %s", err, testCase.name)
		}

		authsAsYaml, err := yaml.Marshal(authconfigs)
		assert.NilError(t, err, "Error parsing authConfigs to yaml in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedAuthConfigs)
		assert.NilError(t, err, "Error parsing exception to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(authsAsYaml), string(expectedAsYaml), "Unexpected authConfigs in testCase %s", testCase.name)

		err = filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
			os.RemoveAll(path)
			return nil
		})
		assert.NilError(t, err, "Error cleaning up in testCase %s", testCase.name)
	}
}
