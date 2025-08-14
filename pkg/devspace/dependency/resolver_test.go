package dependency

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/util"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/remotecache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	fakekube "github.com/loft-sh/devspace/pkg/devspace/kubectl/testing"
	"github.com/loft-sh/devspace/pkg/util/fsutil"
	log "github.com/loft-sh/devspace/pkg/util/log/testing"

	"gotest.tools/assert"
	"k8s.io/client-go/kubernetes/fake"

	yaml "gopkg.in/yaml.v3"
)

type resolverTestCase struct {
	name string

	files           map[string]*latest.Config
	dependencyTasks map[string]*latest.DependencyConfig
	// updateParam          bool
	allowCyclic          bool
	skipIds              bool
	expectedDependencies []Dependency
	expectedErr          string
}

func TestResolver(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
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

	util.DependencyFolderPath = filepath.Join(dir, "dependencyFolder")

	// Delete temp folder
	defer func() {
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
	}()

	testCases := []resolverTestCase{
		{
			name: "No depependency tasks",
		},
		{
			name: "Simple local dependency",
			files: map[string]*latest.Config{
				"dependency1/devspace.yaml": {
					Version: latest.Version,
				},
				"dependency2/devspace.yaml": {
					Version: latest.Version,
				},
			},
			dependencyTasks: map[string]*latest.DependencyConfig{
				"test1": {
					Name: "test1",
					Source: &latest.SourceConfig{
						Path: "dependency1",
					},
				},
				"test2": {
					Name: "test2",
					Source: &latest.SourceConfig{
						Path: "dependency2",
					},
				},
			},
			expectedDependencies: []Dependency{
				{
					dependencyConfig: &latest.DependencyConfig{
						Name: "test1",
						Source: &latest.SourceConfig{
							Path: "dependency1",
						},
					},
					name:         "test1",
					absolutePath: filepath.Join(dir, "dependency1"),
				},
				{
					dependencyConfig: &latest.DependencyConfig{
						Name: "test2",
						Source: &latest.SourceConfig{
							Path: "dependency2",
						},
					},
					name:         "test2",
					absolutePath: filepath.Join(dir, "dependency2"),
				},
			},
		},
		{
			name: "Simple git dependency",
			files: map[string]*latest.Config{
				"dependency1/devspace.yaml": {},
			},
			dependencyTasks: map[string]*latest.DependencyConfig{
				"test": {
					Name: "test",
					Source: &latest.SourceConfig{
						Git:      "https://github.com/devspace-cloud/example-dependency.git",
						Revision: "f8b2aa8cf8ac03238a28e8f78382b214d619893f",
						SubPath:  "mysubpath",
					},
				},
			},
			expectedDependencies: []Dependency{
				{
					absolutePath: filepath.Join(util.DependencyFolderPath, mustGetDependencyID(&latest.DependencyConfig{
						Name: "test",
						Source: &latest.SourceConfig{
							Git:      "https://github.com/devspace-cloud/example-dependency.git",
							Revision: "f8b2aa8cf8ac03238a28e8f78382b214d619893f",
							SubPath:  "mysubpath",
						},
					}), "mysubpath"),
					name: "test",
				},
			},
		},
		{
			name:    "Cyclic allowed dependency",
			skipIds: true,
			files: map[string]*latest.Config{
				"dependency2/devspace.yaml": {
					Version: latest.Version,
					Dependencies: map[string]*latest.DependencyConfig{
						"test2": {
							Name: "test2",
							Source: &latest.SourceConfig{
								Path: "../dependency1",
							},
						},
					},
				},
				"dependency1/devspace.yaml": {
					Version: latest.Version,
					Dependencies: map[string]*latest.DependencyConfig{
						"test1": {
							Name: "test1",
							Source: &latest.SourceConfig{
								Path: "../dependency2",
							},
						},
					},
				},
			},
			dependencyTasks: map[string]*latest.DependencyConfig{
				"test": {
					Name: "test",
					Source: &latest.SourceConfig{
						Path: "dependency1",
					},
				},
			},
			allowCyclic: true,
			expectedDependencies: []Dependency{
				{
					absolutePath: filepath.Join(dir, "dependency1"),
					name:         "test",
				},
			},
		},
	}

	for _, testCase := range testCases {
		for path, content := range testCase.files {
			asYAML, err := yaml.Marshal(content)
			assert.NilError(t, err, "Error parsing config to yaml in testCase %s", testCase.name)
			err = fsutil.WriteToFile(asYAML, path)
			assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
		}

		testConfig := &latest.Config{
			Dependencies: testCase.dependencyTasks,
		}
		generatedConfig := localcache.New(constants.DefaultConfigPath)
		kube := fake.NewSimpleClientset()
		kubeClient := &fakekube.Client{
			Client: kube,
		}

		conf := config.NewConfig(map[string]interface{}{},
			map[string]interface{}{},
			testConfig,
			generatedConfig,
			&remotecache.RemoteCache{},
			map[string]interface{}{},
			constants.DefaultConfigPath)

		devCtx := devspacecontext.NewContext(context.Background(), nil, log.NewFakeLogger()).WithConfig(conf).WithKubeClient(kubeClient)

		testResolver := NewResolver(devCtx, &loader.ConfigOptions{})
		assert.NilError(t, err, "Error creating a resolver in testCase %s", testCase.name)

		dependencies, err := testResolver.Resolve(devCtx, ResolveOptions{})
		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Unexpected error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error from Resolve in testCase %s", testCase.name)
		}

		assert.Equal(t, len(testCase.expectedDependencies), len(dependencies), "Wrong dependency length in testCase %s", testCase.name)
		for index, expected := range testCase.expectedDependencies {
			assert.Equal(t, expected.name, dependencies[index].Name())
			assert.Equal(t, expected.absolutePath, dependencies[index].Path(), "Dependency has wrong local path in testCase %s", testCase.name)
		}

		for path := range testCase.files {
			err = os.Remove(path)
			assert.NilError(t, err, "Error removing file in testCase %s", testCase.name)
		}
		os.RemoveAll(util.DependencyFolderPath) //No error catch because it doesn't need to exist

	}
}

func mustGetDependencyID(config *latest.DependencyConfig) string {
	id, _ := util.GetDependencyID(config.Source)
	return id
}
