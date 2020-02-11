package add

/*
import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"gotest.tools/assert"
)

type addSyncTestCase struct {
	name string

	args       []string
	fakeConfig *latest.Config

	cmd *syncCmd

	expectedErr      string
	expectConfigFile bool
	expectedSync     []*latest.SyncConfig
}

func TestRunAddSync(t *testing.T) {
	testCases := []addSyncTestCase{
		addSyncTestCase{
			name:       "Add empty selector",
			args:       []string{""},
			fakeConfig: &latest.Config{},
			cmd: &syncCmd{
				LocalPath:     "/",
				ContainerPath: "/",
			},
			expectedSync: []*latest.SyncConfig{
				&latest.SyncConfig{
					LabelSelector: map[string]string{
						"app.kubernetes.io/component": "devspace",
					},
					LocalSubPath:  "/",
					ContainerPath: "/",
				},
			},
			expectConfigFile: true,
		},
	}

	log.SetInstance(&log.DiscardLogger{PanicOnExit: true})

	for _, testCase := range testCases {
		testRunAddSync(t, testCase)
	}
}

func testRunAddSync(t *testing.T, testCase addSyncTestCase) {
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

	isDeploymentsNil := testCase.fakeConfig == nil || testCase.fakeConfig.Deployments == nil
	loader.SetFakeConfig(testCase.fakeConfig)
	if isDeploymentsNil && testCase.fakeConfig != nil {
		testCase.fakeConfig.Deployments = nil
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

	if testCase.cmd == nil {
		testCase.cmd = &syncCmd{}
	}
	if testCase.cmd.GlobalFlags == nil {
		testCase.cmd.GlobalFlags = &flags.GlobalFlags{}
	}

	err = (testCase.cmd).RunAddSync(nil, testCase.args)

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
		return
	}

	config, err := loader.GetBaseConfig(&loader.ConfigOptions{})
	if err != nil {
		log.Fatal(err)
	}

	assert.Equal(t, len(testCase.expectedSync), len(config.Dev.Sync), "Wrong number of selectors in testCase %s", testCase.name)
	for index, sync := range config.Dev.Sync {
		assert.Equal(t, testCase.expectedSync[index].Namespace, sync.Namespace, "Namespace of sync unexpected in testCase %s", testCase.name)
		assert.Equal(t, testCase.expectedSync[index].ContainerName, sync.ContainerName, "Container of sync unexpected in testCase %s", testCase.name)
		assert.Equal(t, testCase.expectedSync[index].LocalSubPath, sync.LocalSubPath, "Local path of sync unexpected in testCase %s", testCase.name)
		assert.Equal(t, testCase.expectedSync[index].ContainerPath, sync.ContainerPath, "Containerpat of sync unexpected in testCase %s", testCase.name)
		assert.Equal(t, testCase.expectedSync[index].WaitInitialSync, sync.WaitInitialSync, "WaitInitialSync of sync unexpected in testCase %s", testCase.name)

		if testCase.expectedSync[index].ExcludePaths == nil {
			testCase.expectedSync[index].ExcludePaths = []string{}
		}
		if sync.ExcludePaths == nil {
			sync.ExcludePaths = []string{}
		}
		for index, excludePath := range sync.ExcludePaths {
			assert.Equal(t, (testCase.expectedSync[index].ExcludePaths)[index], excludePath, "ExcludePaths of sync unexpected in testCase %s", testCase.name)
		}

		if testCase.expectedSync[index].DownloadExcludePaths == nil {
			testCase.expectedSync[index].DownloadExcludePaths = []string{}
		}
		if sync.DownloadExcludePaths == nil {
			sync.DownloadExcludePaths = []string{}
		}
		for index, excludePath := range sync.DownloadExcludePaths {
			assert.Equal(t, (testCase.expectedSync[index].DownloadExcludePaths)[index], excludePath, "DownloadExcludePaths of sync unexpected in testCase %s", testCase.name)
		}

		if testCase.expectedSync[index].UploadExcludePaths == nil {
			testCase.expectedSync[index].UploadExcludePaths = []string{}
		}
		if sync.UploadExcludePaths == nil {
			sync.UploadExcludePaths = []string{}
		}
		for index, excludePath := range sync.UploadExcludePaths {
			assert.Equal(t, (testCase.expectedSync[index].UploadExcludePaths)[index], excludePath, "UploadExcludePaths of sync unexpected in testCase %s", testCase.name)
		}

		if testCase.expectedSync[index].LabelSelector == nil {
			testCase.expectedSync[index].LabelSelector = map[string]string{}
		}
		assert.Equal(t, len(testCase.expectedSync[index].LabelSelector), len(sync.LabelSelector), "Unexpected labelselector length in in testCase %s", testCase.name)
		for key, value := range testCase.expectedSync[index].LabelSelector {
			assert.Equal(t, sync.LabelSelector[key], value, "Unexpected labelselector value of key %s in testCase %s", key, testCase.name)
		}
	}

	err = os.Remove(constants.DefaultConfigPath)
	assert.Equal(t, !os.IsNotExist(err), testCase.expectConfigFile, "Unexpectedly saved or not saved in testCase %s", testCase.name)
}*/
