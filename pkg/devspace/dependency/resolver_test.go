package dependency

import (
	"fmt"
	"github.com/devspace-cloud/devspace/pkg/devspace/dependency/util"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	fakekube "github.com/devspace-cloud/devspace/pkg/devspace/kubectl/testing"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"

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
					Source: &latest.SourceConfig{
						Path: "dependency1",
					},
				},
				&latest.DependencyConfig{
					Source: &latest.SourceConfig{
						Path: "dependency2",
					},
				},
			},
			expectedDependencies: []Dependency{
				Dependency{
					ID:        filepath.Join(dir, "dependency1"),
					LocalPath: filepath.Join(dir, "dependency1"),
				},
				Dependency{
					ID:        filepath.Join(dir, "dependency2"),
					LocalPath: filepath.Join(dir, "dependency2"),
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
					Source: &latest.SourceConfig{
						Git:      "https://github.com/devspace-cloud/example-dependency.git",
						Revision: "f8b2aa8cf8ac03238a28e8f78382b214d619893f",
						SubPath:  "mysubpath",
					},
				},
			},
			expectedDependencies: []Dependency{
				Dependency{
					ID:        "https://github.com/devspace-cloud/example-dependency.git@f8b2aa8cf8ac03238a28e8f78382b214d619893f:mysubpath",
					LocalPath: filepath.Join(util.DependencyFolderPath, "84e3f5121aa5a99b3d26752f40e3935f494312ad82d0e85afc9b6e23c762c705", "mysubpath"),
				},
			},
		},
		resolverTestCase{
			name: "Cyclic allowed dependency",
			files: map[string]*latest.Config{
				"dependency1/devspace.yaml": &latest.Config{
					Version: latest.Version,
					Dependencies: []*latest.DependencyConfig{
						&latest.DependencyConfig{
							Source: &latest.SourceConfig{
								Path: "..",
							},
						},
					},
				},
			},
			dependencyTasks: []*latest.DependencyConfig{
				&latest.DependencyConfig{
					Source: &latest.SourceConfig{
						Path: "dependency1",
					},
				},
			},
			allowCyclic: true,
			expectedDependencies: []Dependency{
				Dependency{
					ID:        filepath.Join(dir, "dependency1"),
					LocalPath: filepath.Join(dir, "dependency1"),
				},
			},
		},
		resolverTestCase{
			name: "Cyclic unallowed dependency",
			files: map[string]*latest.Config{
				"dependency1/devspace.yaml": &latest.Config{
					Version: latest.Version,
					Dependencies: []*latest.DependencyConfig{
						&latest.DependencyConfig{
							Source: &latest.SourceConfig{
								Path: "..",
							},
						},
					},
				},
			},
			dependencyTasks: []*latest.DependencyConfig{
				&latest.DependencyConfig{
					Source: &latest.SourceConfig{
						Path: "dependency1",
					},
				},
			},
			expectedErr: fmt.Sprintf("Cyclic dependency found: \n%s\n%s\n%s", filepath.Join(dir, "dependency1"), dir, filepath.Join(dir, "dependency1")),
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
		testResolver, err := NewResolver(testConfig, generatedConfig, kubeClient, testCase.allowCyclic, &loader.ConfigOptions{}, log.Discard)
		assert.NilError(t, err, "Error creating a resolver in testCase %s", testCase.name)

		dependencies, err := testResolver.Resolve(testCase.updateParam)
		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Unexpected error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error from Resolve in testCase %s", testCase.name)
		}

		assert.Equal(t, len(testCase.expectedDependencies), len(dependencies), "Wrong dependency length in testCase %s", testCase.name)
		for index, expected := range testCase.expectedDependencies {
			assert.Equal(t, expected.ID, dependencies[index].ID, "Dependency has wrong id in testCase %s", testCase.name)
			assert.Equal(t, expected.LocalPath, dependencies[index].LocalPath, "Dependency has wrong local path in testCase %s", testCase.name)
		}

		for path := range testCase.files {
			err = os.Remove(path)
			assert.NilError(t, err, "Error removing file in testCase %s", testCase.name)
		}
		os.RemoveAll(util.DependencyFolderPath) //No error catch because it doesn't need to exist

	}
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
			expectedID: "someTagGit@myTag",
		},
		getDependencyIDTestCase{
			name: "git with branch",
			dependency: &latest.DependencyConfig{
				Source: &latest.SourceConfig{
					Git:    "someBranchGit",
					Branch: "myBranch",
				},
			},
			expectedID: "someBranchGit@myBranch",
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
			expectedID: "someRevisionGit@myRevision:mySubPath - profile myProfile",
		},
		getDependencyIDTestCase{
			name: "empty",
			dependency: &latest.DependencyConfig{
				Source: &latest.SourceConfig{},
			},
			expectedID: "",
		},
	}

	for _, testCase := range testCases {
		id := util.GetDependencyID(testCase.baseBath, testCase.dependency.Source, testCase.dependency.Profile)
		assert.Equal(t, testCase.expectedID, id, "Dependency has wrong id in testCase %s", testCase.name)
	}
}
