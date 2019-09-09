package add

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"gotest.tools/assert"
)

type addSelectorTestCase struct {
	name string

	args          []string
	answers       []string
	fakeConfig    *latest.Config
	labelSelector string
	namespace     string

	expectedOutput    string
	expectedPanic     string
	expectConfigFile  bool
	expectedSelectors []*latest.SelectorConfig
}

func TestRunAddSelector(t *testing.T) {
	testCases := []addSelectorTestCase{
		addSelectorTestCase{
			name:          "No devspace config",
			args:          []string{""},
			expectedPanic: "Couldn't find a DevSpace configuration. Please run `devspace init`",
		},
		addSelectorTestCase{
			name:          "Invalid selector",
			args:          []string{""},
			fakeConfig:    &latest.Config{},
			labelSelector: "a=b=c",
			expectedPanic: "Error parsing selectors: Wrong selector format: a=b=c",
		},
		addSelectorTestCase{
			name:           "Add empty selector",
			args:           []string{""},
			fakeConfig:     &latest.Config{},
			expectedOutput: "\nDone Successfully added new service ",
			expectedSelectors: []*latest.SelectorConfig{
				&latest.SelectorConfig{
					LabelSelector: map[string]string{
						"app.kubernetes.io/component": "devspace",
					},
				},
			},
			expectConfigFile: true,
		},
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testRunAddSelector(t, testCase)
	}
}

func testRunAddSelector(t *testing.T, testCase addSelectorTestCase) {
	logOutput = ""

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

	for _, answer := range testCase.answers {
		survey.SetNextAnswer(answer)
	}

	isDeploymentsNil := testCase.fakeConfig == nil || testCase.fakeConfig.Deployments == nil
	configutil.SetFakeConfig(testCase.fakeConfig)
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

		rec := recover()
		if testCase.expectedPanic == "" {
			if rec != nil {
				t.Fatalf("Unexpected panic in testCase %s. Message: %s", testCase.name, rec)
			}
		} else {
			if rec == nil {
				t.Fatalf("Unexpected no panic in testCase %s", testCase.name)
			} else {
				assert.Equal(t, rec, testCase.expectedPanic, "Wrong panic message in testCase %s", testCase.name)
			}
		}
		assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
	}()

	(&selectorCmd{
		LabelSelector: testCase.labelSelector,
		Namespace:     testCase.namespace,
	}).RunAddSelector(nil, testCase.args)

	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)

	config := configutil.GetBaseConfig(context.Background())

	assert.Equal(t, len(testCase.expectedSelectors), len(config.Dev.Selectors), "Wrong number of selectors in testCase %s", testCase.name)
	for index, selector := range config.Dev.Selectors {
		assert.Equal(t, testCase.expectedSelectors[index].Name, selector.Name, "Local port unexpected in testCase %s", testCase.name)
		assert.Equal(t, testCase.expectedSelectors[index].Namespace, selector.Namespace, "Local port unexpected in testCase %s", testCase.name)

		if testCase.expectedSelectors[index].LabelSelector == nil {
			testCase.expectedSelectors[index].LabelSelector = map[string]string{}
		}
		assert.Equal(t, len(testCase.expectedSelectors[index].LabelSelector), len(selector.LabelSelector), "Unexpected labelselector length in selector %s in testCase %s", selector.Name, testCase.name)
		for key, value := range testCase.expectedSelectors[index].LabelSelector {
			assert.Equal(t, selector.LabelSelector[key], value, "Unexpected labelselector value of key %s in selector %s in testCase %s", key, selector.Name, testCase.name)
		}
	}

	err = os.Remove(constants.DefaultConfigPath)
	assert.Equal(t, !os.IsNotExist(err), testCase.expectConfigFile, "Unexpectedly saved or not saved in testCase %s", testCase.name)
}
