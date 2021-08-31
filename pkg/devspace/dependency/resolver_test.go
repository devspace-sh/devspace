package dependency

import (
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
					id:        filepath.Join(dir, "dependency1"),
					localPath: filepath.Join(dir, "dependency1"),
				},
				Dependency{
					id:        filepath.Join(dir, "dependency2"),
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
					id:        "6f6b8b240599fff48a4009c3f78825881deb0306ab09c8ae3e10bf5bef390325",
					localPath: filepath.Join(util.DependencyFolderPath, "84e3f5121aa5a99b3d26752f40e3935f494312ad82d0e85afc9b6e23c762c705", "mysubpath"),
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
							Name: "test",
							Source: &latest.SourceConfig{
								Path: "..",
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
					id:        filepath.Join(dir, "dependency1"),
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
			assert.Equal(t, expected.id, dependencies[index].id, "Dependency has wrong id in testCase %s", testCase.name)
			assert.Equal(t, expected.localPath, dependencies[index].localPath, "Dependency has wrong local path in testCase %s", testCase.name)
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
			expectedID: "2255ee093fea082d8f3cdc8095a723bb0d1104c1b28a017b636ff4aaa806a064",
		},
		getDependencyIDTestCase{
			name: "git with branch",
			dependency: &latest.DependencyConfig{
				Source: &latest.SourceConfig{
					Git:    "someBranchGit",
					Branch: "myBranch",
				},
			},
			expectedID: "6a475ccc660b39e1f8e7b701a206a19e9347457b0ee80910b3fece44a2867598",
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
			expectedID: "947020214d9a41c68c3947e5fdbaf3d0541fca84ceee9a7b7bbf5300fb053f9e",
		},
		getDependencyIDTestCase{
			name: "empty",
			dependency: &latest.DependencyConfig{
				Source: &latest.SourceConfig{},
			},
			expectedID: "5c7bed9afdec33b1d734b7dce7666bc43c5d389373af94a463b1aaebac61f013",
		},
	}

	for _, testCase := range testCases {
		id, err := util.GetDependencyID(testCase.baseBath, testCase.dependency)
		assert.NilError(t, err)
		assert.Equal(t, testCase.expectedID, id, "Dependency has wrong id in testCase %s", testCase.name)
	}
}
