package dependency

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	"gotest.tools/assert"

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
	expectedLog          string
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

	DependencyFolderPath = filepath.Join(dir, "dependencyFolder")

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
			name:        "No depependency tasks",
			expectedLog: "\nWait Resolving dependencies\nDone Resolved 0 dependencies",
		},
		resolverTestCase{
			name: "Simple local dependency",
			files: map[string]*latest.Config{
				"dependency1/devspace.yaml": &latest.Config{},
			},
			dependencyTasks: []*latest.DependencyConfig{
				&latest.DependencyConfig{
					Source: &latest.SourceConfig{
						Path: ptr.String("dependency1"),
					},
				},
			},
			expectedDependencies: []Dependency{
				Dependency{
					ID:        filepath.Join(dir, "dependency1"),
					LocalPath: filepath.Join(dir, "dependency1"),
				},
			},
			expectedLog: "\nWait Resolving dependencies\nDone Resolved 1 dependencies",
		},
		resolverTestCase{
			name: "Simple git dependency",
			files: map[string]*latest.Config{
				"dependency1/devspace.yaml": &latest.Config{},
			},
			dependencyTasks: []*latest.DependencyConfig{
				&latest.DependencyConfig{
					Source: &latest.SourceConfig{
						Git:      ptr.String("https://github.com/devspace-cloud/example-dependency.git"),
						Revision: ptr.String("9e0a7b806035b92d98eb1a71f650f695a2dd8b24"),
						SubPath:  ptr.String("mysubpath"),
					},
				},
			},
			expectedDependencies: []Dependency{
				Dependency{
					ID:        "https://github.com/devspace-cloud/example-dependency.git@9e0a7b806035b92d98eb1a71f650f695a2dd8b24:mysubpath",
					LocalPath: filepath.Join(DependencyFolderPath, "9fb6955fb5f5fcce7a3277f4b3e8ac447aeba1d3ac8e4f4107be7e0ac7d3ce5f", "mysubpath"),
				},
			},
			expectedLog: "\nWait Resolving dependencies\nDone Pulled https://github.com/devspace-cloud/example-dependency.git@9e0a7b806035b92d98eb1a71f650f695a2dd8b24:mysubpath\nDone Resolved 1 dependencies",
		},
		resolverTestCase{
			name: "Cyclic allowed dependency",
			files: map[string]*latest.Config{
				"dependency1/devspace.yaml": &latest.Config{
					Dependencies: &[]*latest.DependencyConfig{
						&latest.DependencyConfig{
							Source: &latest.SourceConfig{
								Path: ptr.String(".."),
							},
						},
					},
				},
			},
			dependencyTasks: []*latest.DependencyConfig{
				&latest.DependencyConfig{
					Source: &latest.SourceConfig{
						Path: ptr.String("dependency1"),
					},
					Namespace: ptr.String("someNamespace"),
				},
			},
			allowCyclic: true,
			expectedDependencies: []Dependency{
				Dependency{
					ID:        filepath.Join(dir, "dependency1"),
					LocalPath: filepath.Join(dir, "dependency1"),
				},
			},
			expectedLog: "\nWait Resolving dependencies\nDone Resolved 1 dependencies",
		},
		resolverTestCase{
			name: "Cyclic unallowed dependency",
			files: map[string]*latest.Config{
				"dependency1/devspace.yaml": &latest.Config{
					Dependencies: &[]*latest.DependencyConfig{
						&latest.DependencyConfig{
							Source: &latest.SourceConfig{
								Path: ptr.String(".."),
							},
						},
					},
				},
			},
			dependencyTasks: []*latest.DependencyConfig{
				&latest.DependencyConfig{
					Source: &latest.SourceConfig{
						Path: ptr.String("dependency1"),
					},
				},
			},
			expectedErr: fmt.Sprintf("Cyclic dependency found: \n%s\n%s\n%s", filepath.Join(dir, "dependency1"), dir, filepath.Join(dir, "dependency1")),
			expectedLog: "\nWait Resolving dependencies",
		},
	}

	for _, testCase := range testCases {
		for path, content := range testCase.files {
			asYAML, err := yaml.Marshal(content)
			assert.NilError(t, err, "Error parsing config to yaml in testCase %s", testCase.name)
			err = fsutil.WriteToFile(asYAML, path)
			assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
		}

		logOutput = ""

		testConfig := &latest.Config{}
		generatedConfig := &generated.Config{}
		testResolver, err := NewResolver(testConfig, generatedConfig, testCase.allowCyclic, &testLogger{})
		assert.NilError(t, err, "Error creating a resolver in testCase %s", testCase.name)

		dependencies, err := testResolver.Resolve(testCase.dependencyTasks, testCase.updateParam)

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
		os.RemoveAll(DependencyFolderPath) //No error catch because it doesn't need to exist

		assert.Equal(t, logOutput, testCase.expectedLog, "Unexpected output in testCase %s", testCase.name)

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
