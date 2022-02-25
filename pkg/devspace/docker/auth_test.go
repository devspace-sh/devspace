package docker

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/registry"
	dockerclient "github.com/docker/docker/client"
	"github.com/loft-sh/devspace/pkg/util/fsutil"
	"gopkg.in/yaml.v3"

	"gotest.tools/assert"
)

type fakeDockerClient struct {
	dockerclient.Client
}

func (f *fakeDockerClient) Info(ctx context.Context) (types.Info, error) {
	return types.Info{
		IndexServerAddress: "IndexServerAddress",
	}, nil
}

func (f *fakeDockerClient) Ping(ctx context.Context) (types.Ping, error) {
	return types.Ping{}, nil
}

func (f *fakeDockerClient) RegistryLogin(ctx context.Context, auth types.AuthConfig) (registry.AuthenticateOKBody, error) {
	identityToken := ""
	if auth.Password == "useToken" {
		identityToken = "someToken"
	}
	return registry.AuthenticateOKBody{
		IdentityToken: identityToken,
	}, nil
}

func (f *fakeDockerClient) ImageList(ctx context.Context, options types.ImageListOptions) ([]types.ImageSummary, error) {
	return []types.ImageSummary{
		{
			ID: "deleteThis",
		},
	}, nil
}

func (f *fakeDockerClient) ImageRemove(ctx context.Context, image string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
	return []types.ImageDeleteResponseItem{
		{
			Deleted:  "deleteThis",
			Untagged: "deleteThis",
		},
	}, nil
}

type getRegistryEndpointTestCase struct {
	name string

	registryURL string

	expectedIsDefault bool
	expectedEndpoint  string
	expectedErr       bool
}

func TestGetRegistryEndpoint(t *testing.T) {
	testCases := []getRegistryEndpointTestCase{
		{
			name:              "Use auth server",
			expectedIsDefault: true,
			expectedEndpoint:  "IndexServerAddress",
		},
		{
			name:              "Use custom server",
			registryURL:       "custom",
			expectedIsDefault: false,
			expectedEndpoint:  "custom",
		},
	}

	for _, testCase := range testCases {
		client := &client{
			CommonAPIClient: &fakeDockerClient{},
		}

		isDefault, endpoint, err := client.GetRegistryEndpoint(testCase.registryURL)

		if !testCase.expectedErr {
			assert.NilError(t, err, "Unexpected error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected error %v in testCase %s", err, testCase.name)
		}

		assert.Equal(t, isDefault, testCase.expectedIsDefault, "Unexpected isDefault bool in testCase %s", testCase.name)
		assert.Equal(t, endpoint, testCase.expectedEndpoint, "Unexpected endpoint in testCase %s", testCase.name)
	}
}

type getAuthConfigTestCase struct {
	name string

	files                 map[string]interface{}
	registryURL           string
	checkCredentialsStore bool

	expectedAuthConfig *types.AuthConfig
	expectedErr        bool
}

func TestGetAuthConfig(t *testing.T) {
	testCases := []getAuthConfigTestCase{
		{
			name:                  "Use default server",
			checkCredentialsStore: true,
			expectedAuthConfig: &types.AuthConfig{
				ServerAddress: "IndexServerAddress",
			},
		},
		{
			name:                  "Use custom server",
			registryURL:           "http://custom",
			checkCredentialsStore: true,
			expectedAuthConfig: &types.AuthConfig{
				ServerAddress: "custom",
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

		client := &client{
			CommonAPIClient: &fakeDockerClient{},
		}

		auth, err := client.GetAuthConfig(testCase.registryURL, testCase.checkCredentialsStore)

		if !testCase.expectedErr {
			assert.NilError(t, err, "Unexpected error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected error %v in testCase %s", err, testCase.name)
		}

		authAsYaml, err := yaml.Marshal(auth)
		assert.NilError(t, err, "Error parsing authConfig to yaml in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedAuthConfig)
		assert.NilError(t, err, "Error parsing exception to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(authAsYaml), string(expectedAsYaml), "Unexpected authConfig in testCase %s", testCase.name)

		err = filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
			os.RemoveAll(path)
			return nil
		})
		assert.NilError(t, err, "Error cleaning up in testCase %s", testCase.name)
	}
}

type loginTestCase struct {
	name string

	files                 map[string]interface{}
	registryURL           string
	user                  string
	password              string
	checkCredentialsStore bool
	saveAuthConfig        bool
	relogin               bool

	expectedAuthConfig *types.AuthConfig
	expectedErr        bool
}

func TestLogin(t *testing.T) {
	testCases := []loginTestCase{
		{
			name:                  "Use default server",
			checkCredentialsStore: true,
			saveAuthConfig:        true,
			user:                  "user",
			password:              "useToken",
			expectedAuthConfig: &types.AuthConfig{
				ServerAddress: "IndexServerAddress",
				Username:      "user",
				IdentityToken: "someToken",
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

		client := &client{
			CommonAPIClient: &fakeDockerClient{},
		}

		auth, err := client.Login(testCase.registryURL, testCase.user, testCase.password, testCase.checkCredentialsStore, testCase.saveAuthConfig, testCase.relogin)
		if !testCase.expectedErr {
			assert.NilError(t, err, "Unexpected error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected error %v in testCase %s", err, testCase.name)
		}

		authAsYaml, err := yaml.Marshal(auth)
		assert.NilError(t, err, "Error parsing authConfig to yaml in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedAuthConfig)
		assert.NilError(t, err, "Error parsing exception to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(authAsYaml), string(expectedAsYaml), "Unexpected authConfig in testCase %s", testCase.name)

		err = filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
			os.RemoveAll(path)
			return nil
		})
		assert.NilError(t, err, "Error cleaning up in testCase %s", testCase.name)
	}
}
