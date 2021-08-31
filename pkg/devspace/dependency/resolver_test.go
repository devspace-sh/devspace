package dependency

import (
	"github.com/loft-sh/devspace/pkg/util/hash"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/util"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	fakekube "github.com/loft-sh/devspace/pkg/devspace/kubectl/testing"
	"github.com/loft-sh/devspace/pkg/util/fsutil"
	"github.com/loft-sh/devspace/pkg/util/log"

	"gotest.tools/assert"
	"k8s.io/client-go/kubernetes/fake"

	yaml "gopkg.in/yaml.v2"
)

type resolverTestCase struct {
	name string

	files           map[string]*latest.Config
	dependencyTasks []*latest.DependencyConfig
	updateParam     bool
	allowCyclic     bool

	skipIds              bool
	expectedDependencies []Dependency
	expectedErr          string
}

func TestResolver(t *testing.T) {
	dir, err := ioutil.TempDir("", "testFolder")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}
	dir, err = filepath.EvalSymlinks(dir)
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
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}()

	testCases := []resolverTestCase{
		resolverTestCase{
			name: "No depependency tasks",
		},
		resolverTestCase{
			name: "Simple local dependency",
			files: map[string]*latest.Config{
				"dependency1/devspace.yaml": &latest.Config{
					Version: latest.Version,
				},
				"dependency2/devspace.yaml": &latest.Config{
					Version: latest.Version,
				},
			},
			dependencyTasks: []*latest.DependencyConfig{
				&latest.DependencyConfig{
					Name: "test",
					Source: &latest.SourceConfig{
						Path: "dependency1",
					},
				},
				&latest.DependencyConfig{
					Name: "test",
					Source: &latest.SourceConfig{
						Path: "dependency2",
					},
				},
			},
			expectedDependencies: []Dependency{
				Dependency{
					localPath: filepath.Join(dir, "dependency1"),
				},
				Dependency{
					localPath: filepath.Join(dir, "dependency2"),
				},
			},
		},
		resolverTestCase{
			name: "Simple git dependency",
			files: map[string]*latest.Config{
				"dependency1/devspace.yaml": &latest.Config{},
			},
			dependencyTasks: []*latest.DependencyConfig{
				&latest.DependencyConfig{
					Name: "test",
					Source: &latest.SourceConfig{
						Git:      "https://github.com/devspace-cloud/example-dependency.git",
						Revision: "f8b2aa8cf8ac03238a28e8f78382b214d619893f",
						SubPath:  "mysubpath",
					},
				},
			},
			expectedDependencies: []Dependency{
				Dependency{
					localPath: filepath.Join(util.DependencyFolderPath, hash.String(mustGetDependencyID(dir, &latest.DependencyConfig{
						Name: "test",
						Source: &latest.SourceConfig{
							Git:      "https://github.com/devspace-cloud/example-dependency.git",
							Revision: "f8b2aa8cf8ac03238a28e8f78382b214d619893f",
							SubPath:  "mysubpath",
						},
					})), "mysubpath"),
				},
			},
		},
		resolverTestCase{
			name:    "Cyclic allowed dependency",
			skipIds: true,
			files: map[string]*latest.Config{
				"dependency2/devspace.yaml": &latest.Config{
					Version: latest.Version,
					Dependencies: []*latest.DependencyConfig{
						&latest.DependencyConfig{
							Name: "test",
							Source: &latest.SourceConfig{
								Path: "../dependency1",
							},
						},
					},
				},
				"dependency1/devspace.yaml": &latest.Config{
					Version: latest.Version,
					Dependencies: []*latest.DependencyConfig{
						&latest.DependencyConfig{
							Name: "test",
							Source: &latest.SourceConfig{
								Path: "../dependency2",
							},
						},
					},
				},
			},
			dependencyTasks: []*latest.DependencyConfig{
				&latest.DependencyConfig{
					Name: "test",
					Source: &latest.SourceConfig{
						Path: "dependency1",
					},
				},
			},
			allowCyclic: true,
			expectedDependencies: []Dependency{
				Dependency{
					localPath: filepath.Join(dir, "dependency2"),
				},
				Dependency{
					localPath: filepath.Join(dir, "dependency1"),
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
		generatedConfig := &generated.Config{}
		kube := fake.NewSimpleClientset()
		kubeClient := &fakekube.Client{
			Client: kube,
		}
		testResolver := NewResolver(config.NewConfig(nil, testConfig, generatedConfig, map[string]interface{}{}, constants.DefaultConfigPath), kubeClient, &loader.ConfigOptions{}, log.Discard)
		assert.NilError(t, err, "Error creating a resolver in testCase %s", testCase.name)

		dependencies, err := testResolver.Resolve(testCase.updateParam)
		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Unexpected error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error from Resolve in testCase %s", testCase.name)
		}

		assert.Equal(t, len(testCase.expectedDependencies), len(dependencies), "Wrong dependency length in testCase %s", testCase.name)
		for index, expected := range testCase.expectedDependencies {
			if testCase.skipIds == false {
				id, _ := util.GetDependencyID(dir, testCase.dependencyTasks[index])
				assert.Equal(t, id, dependencies[index].id, "Dependency has wrong id in testCase %s", testCase.name)
			}
			assert.Equal(t, expected.localPath, dependencies[index].localPath, "Dependency has wrong local path in testCase %s", testCase.name)
		}

		for path := range testCase.files {
			err = os.Remove(path)
			assert.NilError(t, err, "Error removing file in testCase %s", testCase.name)
		}
		os.RemoveAll(util.DependencyFolderPath) //No error catch because it doesn't need to exist

	}
}

func mustGetDependencyID(basePath string, config *latest.DependencyConfig) string {
	id, _ := util.GetDependencyID(basePath, config)
	return id
}

func includes(arr []string, needle string) bool {
	for _, suspect := range arr {
		if suspect == needle {
			return true
		}
	}
	return false
}

type getDependencyIDTestCase struct {
	name string

	baseBath   string
	dependency *latest.DependencyConfig

	expectedID string
}

func TestGetDependencyID(t *testing.T) {
	testCases := []getDependencyIDTestCase{
		getDependencyIDTestCase{
			name: "git with tag",
			dependency: &latest.DependencyConfig{
				Source: &latest.SourceConfig{
					Git: "someTagGit",
					Tag: "myTag",
				},
			},
			expectedID: "e8fb9810c53ca0986d12ec5d078e38659a1700425a292cefe4f77bffa351667c",
		},
		getDependencyIDTestCase{
			name: "git with branch",
			dependency: &latest.DependencyConfig{
				Source: &latest.SourceConfig{
					Git:    "someBranchGit",
					Branch: "myBranch",
				},
			},
			expectedID: "9a5ed87e8fec108a03b592058f7eec3a0b1c9fe431cfe1d03a5d37333fb07b2d",
		},
		getDependencyIDTestCase{
			name: "git with revision, subpath and profile",
			dependency: &latest.DependencyConfig{
				Source: &latest.SourceConfig{
					Git:      "someRevisionGit",
					Revision: "myRevision",
					SubPath:  "mySubPath",
				},
				Profile: "myProfile",
			},
			expectedID: "bb783d78de53d3bcb1533d239a3d1d685070f22b9f25e5c487a83425be586900",
		},
		getDependencyIDTestCase{
			name: "empty",
			dependency: &latest.DependencyConfig{
				Source: &latest.SourceConfig{},
			},
			expectedID: "cc4af1ccc6f0bba9d05b89a8ac9bfdca135653f86d9535756b8c219bc7fdd9a1",
		},
	}

	for _, testCase := range testCases {
		id, err := util.GetDependencyID(testCase.baseBath, testCase.dependency)
		assert.NilError(t, err)
		assert.Equal(t, testCase.expectedID, id, "Dependency has wrong id in testCase %s", testCase.name)
	}
}
