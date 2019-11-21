package update

/*
import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/v1alpha1"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type updateConfigTestCase struct {
	name string

	globalFlags flags.GlobalFlags
	files       map[string]interface{}

	expectedConfig interface{}
	expectedErr    string
}

func TestRunUpdateConfig(t *testing.T) {
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
		//Delete temp folder
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}()

	testCases := []updateConfigTestCase{
		updateConfigTestCase{
			name: "Safe with profiles",
			files: map[string]interface{}{
				constants.DefaultConfigPath: latest.Config{
					Version: latest.Version,
					Profiles: []*latest.ProfileConfig{
						&latest.ProfileConfig{},
					},
				},
			},
			expectedConfig: latest.Config{
				Version: latest.Version,
				Dev:     &latest.DevConfig{},
			},
		},
		updateConfigTestCase{
			name: "Old version",
			files: map[string]interface{}{
				constants.DefaultConfigPath: v1alpha1.Config{
					Version: ptr.String(v1alpha1.Version),
					DevSpace: &v1alpha1.DevSpaceConfig{
						Services: &[]*v1alpha1.ServiceConfig{
							&v1alpha1.ServiceConfig{
								Name: ptr.String("terminalService"),
							},
						},
						Terminal: &v1alpha1.Terminal{
							Disabled:      ptr.Bool(true),
							Service:       ptr.String("terminalService"),
							ResourceType:  ptr.String("terminalRT"),
							LabelSelector: &map[string]*string{"hello": ptr.String("World")},
							Namespace:     ptr.String("someNS"),
							ContainerName: ptr.String("someContainer"),
							Command:       &[]*string{ptr.String("myCommand")},
						},
					},
				},
			},
		},
	}

	log.SetInstance(&log.DiscardLogger{PanicOnExit: true})

	for _, testCase := range testCases {
		testRunUpdateConfig(t, testCase)
	}
}

func testRunUpdateConfig(t *testing.T, testCase updateConfigTestCase) {
	for path, content := range testCase.files {
		asYAML, err := yaml.Marshal(content)
		assert.NilError(t, err, "Error parsing config to yaml in testCase %s", testCase.name)
		err = fsutil.WriteToFile(asYAML, path)
		assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
	}

	loader.ResetConfig()

	err := (&configCmd{
		GlobalFlags: &testCase.globalFlags,
	}).RunConfig(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}

	err = filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
		os.RemoveAll(path)
		return nil
	})
	assert.NilError(t, err, "Error cleaning up in testCase %s", testCase.name)
}
*/
