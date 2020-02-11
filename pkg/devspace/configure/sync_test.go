package configure

/*
import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"gotest.tools/assert"
)

type addSyncPathTestCase struct {
	name string

	fakeConfig               *latest.Config
	localPathParam           string
	containerPathParam       string
	namespace                string
	labelSelectorParam       string
	excludedPathsStringParam string

	expectedErr          string
	expectedSyncInConfig []*latest.SyncConfig
}

func TestAddSyncPath(t *testing.T) {
	testCases := []addSyncPathTestCase{
		addSyncPathTestCase{
			name:               "Add sync path with wrong containerPath",
			containerPathParam: " ",
			expectedErr:        "ContainerPath (--container) must start with '/'. Info: There is an issue with MINGW based terminals like git bash",
		},
		addSyncPathTestCase{
			name:                     "Add sync path with success",
			fakeConfig:               &latest.Config{},
			containerPathParam:       "/containerPath",
			excludedPathsStringParam: "./ExcludeThis",
			labelSelectorParam:       "Hello=World",
			expectedSyncInConfig: []*latest.SyncConfig{
				&latest.SyncConfig{
					LabelSelector: map[string]string{"Hello": "World"},
					ContainerPath: "/containerPath",
					LocalSubPath:  "",
					ExcludePaths:  []string{"./ExcludeThis"},
					Namespace:     "",
				},
			},
		},
	}

	//Make temporary test dir
	dir, err := ioutil.TempDir("", "testDir")
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

	// 8. Delete temp folder
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

	for _, testCase := range testCases {
		if testCase.fakeConfig == nil {
			testCase.fakeConfig = &latest.Config{}
		} else {
			loader.SetFakeConfig(testCase.fakeConfig)
		}

		err = AddSyncPath(testCase.fakeConfig, testCase.localPathParam, testCase.containerPathParam, testCase.namespace, testCase.labelSelectorParam, testCase.excludedPathsStringParam)
		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error adding sync path in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error from AddSyncPath in testCase %s", testCase.name)
		}

		assert.Equal(t, len(testCase.fakeConfig.Dev.Sync), len(testCase.expectedSyncInConfig), "Wrong number of syncs after adding in testCase %s", testCase.name)
		for index := range testCase.expectedSyncInConfig {
			for key, value := range testCase.expectedSyncInConfig[index].LabelSelector {
				assert.Equal(t, value, testCase.fakeConfig.Dev.Sync[index].LabelSelector[key], "Wrong labelSelectorMap in added sync in testCase %s", testCase.name)
			}
			assert.Equal(t, testCase.expectedSyncInConfig[index].ContainerPath, testCase.fakeConfig.Dev.Sync[index].ContainerPath, "Wrong containerPath in added sync in testCase %s", testCase.name)
			assert.Equal(t, testCase.expectedSyncInConfig[index].LocalSubPath, testCase.fakeConfig.Dev.Sync[index].LocalSubPath, "Wrong LocalSubPath in added sync in testCase %s", testCase.name)
			for excludePathIndex, excludePath := range testCase.expectedSyncInConfig[index].ExcludePaths {
				assert.Equal(t, excludePath, testCase.fakeConfig.Dev.Sync[index].ExcludePaths[excludePathIndex], "Wrong excluded path in added sync in testCase %s", testCase.name)
			}
			assert.Equal(t, testCase.expectedSyncInConfig[index].Namespace, testCase.fakeConfig.Dev.Sync[index].Namespace, "Wrong Namespace in added sync in testCase %s", testCase.name)
		}
	}
}

type removeSyncPathTestCase struct {
	name string

	fakeConfig         *latest.Config
	removeAllParam     bool
	localPathParam     string
	containerPathParam string
	labelSelectorParam string

	expectedErr                string
	expectedSyncPathLocalPaths []string
}

func TestRemoveSyncPath(t *testing.T) {
	testCases := []removeSyncPathTestCase{
		removeSyncPathTestCase{
			name:                       "No flag",
			fakeConfig:                 nil, //default config has two syncPaths
			expectedErr:                "You have to specify at least one of the supported flags",
			expectedSyncPathLocalPaths: []string{"somePath", "someOtherPath"},
		},
		removeSyncPathTestCase{
			name:           "Remove all",
			fakeConfig:     nil, //default config has two syncPaths
			removeAllParam: true,
		},
		removeSyncPathTestCase{
			name:                       "Remove one by local file",
			fakeConfig:                 nil, //default config has two syncPaths
			localPathParam:             "somePath",
			expectedSyncPathLocalPaths: []string{"someOtherPath"},
		},
		removeSyncPathTestCase{
			name:                       "Remove one by labelSelectorMap",
			fakeConfig:                 nil, //default config has two syncPaths
			labelSelectorParam:         "index=secound",
			expectedSyncPathLocalPaths: []string{"somePath"},
		},
	}

	//Make temporary test dir
	dir, err := ioutil.TempDir("", "testDir")
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

	// 8. Delete temp folder
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

	for _, testCase := range testCases {
		if testCase.fakeConfig == nil {
			testCase.fakeConfig = &latest.Config{
				Dev: &latest.DevConfig{
					Sync: []*latest.SyncConfig{
						&latest.SyncConfig{
							LocalSubPath:  "somePath",
							ContainerPath: "someContainerPath",
							LabelSelector: map[string]string{
								"index": "first",
							},
						},
						&latest.SyncConfig{
							LocalSubPath:  "someOtherPath",
							ContainerPath: "someOtherContainerPath",
							LabelSelector: map[string]string{
								"index": "secound",
							},
						},
					},
				},
			} //default config
		}
		loader.SetFakeConfig(testCase.fakeConfig)
		config, err := loader.GetBaseConfig(&loader.ConfigOptions{})
		if err != nil {
			log.Fatal(err)
		}

		err = RemoveSyncPath(config, testCase.removeAllParam, testCase.localPathParam, testCase.containerPathParam, testCase.labelSelectorParam)
		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error initializing namespace in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error from initializing namespace in testCase %s", testCase.name)
		}

		assert.Equal(t, len(testCase.expectedSyncPathLocalPaths), len(testCase.fakeConfig.Dev.Sync), "Wrong number of remaining syncPaths in testCase %s", testCase.name)
	OUTER:
		for _, expectedLocalPath := range testCase.expectedSyncPathLocalPaths {
			for _, syncPath := range testCase.fakeConfig.Dev.Sync {
				if syncPath.LocalSubPath == expectedLocalPath {
					continue OUTER
				}
			}
			t.Fatalf("Expected remaining LocalPath %s not found in sync paths", expectedLocalPath)
		}
	}
}*/
