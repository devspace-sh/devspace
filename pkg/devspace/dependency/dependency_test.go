package dependency

import(
	"io/ioutil"
	"os"
	"testing"
	
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
)

func TestDependency(t *testing.T) {
	dir, err := ioutil.TempDir("", "testFolder")
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

	// Delete temp folder
	defer os.Chdir(wdBackup)
	defer os.RemoveAll(dir)

	err = fsutil.WriteToFile([]byte(""), "devspace.yaml")
	if err != nil {
		t.Fatalf("Error writing file: %v", err)
	}

	dependencyTasks := []*latest.DependencyConfig{
		&latest.DependencyConfig{
			Source: &latest.SourceConfig{},
			Config: ptr.String("devspace.yaml"),
		},
	}

	testConfig := &latest.Config{
		Dependencies: &dependencyTasks,
	}
	generatedConfig := &generated.Config{}
	err = UpdateAll(&latest.Config{}, generatedConfig, true, &log.DiscardLogger{})
	if err != nil {
		t.Fatalf("Error updating all dependencies with empty config: %v", err)
	}

	err = UpdateAll(testConfig, generatedConfig, true, &log.DiscardLogger{})
	if err != nil {
		t.Fatalf("Error updating all dependencies: %v", err)
	}

	err = DeployAll(&latest.Config{}, generatedConfig, true, true, true, true, true, true, &log.DiscardLogger{})
	if err != nil {
		t.Fatalf("Error deploying all dependencies with empty config: %v", err)
	}

	err = DeployAll(testConfig, generatedConfig, true, true, true, true, true, true, &log.DiscardLogger{})
	if err != nil {
		t.Fatalf("Error deploying all dependencies: %v", err)
	}

}
