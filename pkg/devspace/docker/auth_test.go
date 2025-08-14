package docker

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	dockerregistry "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/api/types/system"
	dockerclient "github.com/docker/docker/client"
	"github.com/loft-sh/devspace/pkg/util/fsutil"
	"gopkg.in/yaml.v3"
	
	"gotest.tools/assert"
)

type fakeDockerClient struct {
	dockerclient.Client
}

func (f *fakeDockerClient) Info(ctx context.Context) (system.Info, error) {
	return system.Info{
		IndexServerAddress: "IndexServerAddress",
	}, nil
}

func (f *fakeDockerClient) Ping(ctx context.Context) (types.Ping, error) {
	return types.Ping{}, nil
}

func (f *fakeDockerClient) RegistryLogin(ctx context.Context, auth dockerregistry.AuthConfig) (dockerregistry.AuthenticateOKBody, error) {
	identityToken := ""
	if auth.Password == "useToken" {
		identityToken = "someToken"
	}
	return dockerregistry.AuthenticateOKBody{
		IdentityToken: identityToken,
	}, nil
}

func (f *fakeDockerClient) ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error) {
	return []image.Summary{
		{
			ID: "deleteThis",
		},
	}, nil
}

func (f *fakeDockerClient) ImageRemove(ctx context.Context, img string, options image.RemoveOptions) ([]image.DeleteResponse, error) {
	return []image.DeleteResponse{
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
			APIClient: &fakeDockerClient{},
		}
		
		isDefault, endpoint, err := client.GetRegistryEndpoint(context.Background(), testCase.registryURL)
		
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
	
	expectedAuthConfig *dockerregistry.AuthConfig
	expectedErr        bool
}

func TestGetAuthConfig(t *testing.T) {
	testCases := []getAuthConfigTestCase{
		{
			name:                  "Use default server",
			checkCredentialsStore: true,
			expectedAuthConfig: &dockerregistry.AuthConfig{
				ServerAddress: "IndexServerAddress",
			},
		},
		{
			name:                  "Use custom server",
			registryURL:           "http://custom",
			checkCredentialsStore: true,
			expectedAuthConfig: &dockerregistry.AuthConfig{
				ServerAddress: "custom",
			},
		},
	}
	
	dir := t.TempDir()
	
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
			APIClient: &fakeDockerClient{},
		}
		
		auth, err := client.GetAuthConfig(context.Background(), testCase.registryURL, testCase.checkCredentialsStore)
		
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
	
	expectedAuthConfig *dockerregistry.AuthConfig
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
			expectedAuthConfig: &dockerregistry.AuthConfig{
				ServerAddress: "IndexServerAddress",
				Username:      "user",
				IdentityToken: "someToken",
			},
		},
	}
	
	dir := t.TempDir()
	
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
			APIClient: &fakeDockerClient{},
		}
		
		auth, err := client.Login(context.Background(), testCase.registryURL, testCase.user, testCase.password, testCase.checkCredentialsStore, testCase.saveAuthConfig, testCase.relogin)
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
