package dependency

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/hash"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	"gotest.tools/assert"
)

func TestResolver(t *testing.T) {
	t.Skip("Skipped for now")

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

	err = fsutil.WriteToFile([]byte(""), "devspace.yaml")
	if err != nil {
		t.Fatalf("Error writing file: %v", err)
	}
	err = fsutil.WriteToFile([]byte(""), "someDir/devspace.yaml")
	if err != nil {
		t.Fatalf("Error writing file: %v", err)
	}
	gitPath := "https://github.com/devspace-cloud/quickstart-nodejs.git"
	gitDepPath := filepath.Join(DependencyFolderPath, hash.String(gitPath))
	err = fsutil.WriteToFile([]byte(""), filepath.Join(gitDepPath, "devspace.yaml"))
	if err != nil {
		t.Fatalf("Error writing file: %v", err)
	}

	testConfig := &latest.Config{}
	generatedConfig := &generated.Config{}
	testResolver, err := NewResolver(testConfig, generatedConfig, true, &log.DiscardLogger{})
	if err != nil {
		t.Fatalf("Error creating a test resolver: %v", err)
	}

	dependencyTasks := []*latest.DependencyConfig{
		&latest.DependencyConfig{
			Source: &latest.SourceConfig{},
			Config: ptr.String("devspace.yaml"),
		},
		&latest.DependencyConfig{
			Source: &latest.SourceConfig{
				Path: ptr.String("someDir"),
			},
			Config: ptr.String("someDir/devspace.yaml"),
		},
		&latest.DependencyConfig{
			Source: &latest.SourceConfig{
				Git: ptr.String(gitPath),
			},
			Config: ptr.String("someDir/devspace.yaml"),
		},
	}

	dependencies, err := testResolver.Resolve(dependencyTasks, false)
	if err != nil {
		t.Fatalf("Error resolving dependencies: %v", err)
	}
	assert.Equal(t, 3, len(dependencies), "Wrong dependency length")
	assert.Equal(t, "", dependencies[0].ID, "First dependency has wrong id")
	assert.Equal(t, "", dependencies[0].LocalPath, "First dependency has wrong local path")
	assert.Equal(t, filepath.Join(dir, "someDir"), dependencies[1].ID, "Secound dependency has wrong id")
	assert.Equal(t, "", dependencies[0].LocalPath, "Secound dependency has wrong local path")
	assert.Equal(t, gitPath, dependencies[2].ID, "Third dependency has wrong id")
	assert.Equal(t, gitDepPath, dependencies[2].LocalPath, "Third dependency has wrong local path")
}
