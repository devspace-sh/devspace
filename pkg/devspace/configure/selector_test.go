package configure

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	v1 "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"gotest.tools/assert"
)

type addSelectorTestCase struct {
	name string

	selectorName    string
	labelSelector   string
	namespace       string
	save            bool
	selectorsBefore []*v1.SelectorConfig

	expectedErr                 string
	expectedSelectorsAfterwards []v1.SelectorConfig
	expectedSavedConfig         bool
}

func TestAddSelector(t *testing.T) {
	testCases := []addSelectorTestCase{
		addSelectorTestCase{
			name: "Empty input",
			expectedSelectorsAfterwards: []v1.SelectorConfig{
				v1.SelectorConfig{
					LabelSelector: map[string]string{
						"app.kubernetes.io/component": "devspace",
					},
				},
			},
		},
		addSelectorTestCase{
			name:          "Bad selector map",
			labelSelector: "this=does=not=work",
			save:          true,
			expectedErr:   "Error parsing selectors: Wrong selector format: this=does=not=work",
		},
		addSelectorTestCase{
			name:         "Save afterwards",
			selectorName: "newSelector",
			namespace:    "namespace-for-new-selector",
			save:         true,
			selectorsBefore: []*v1.SelectorConfig{
				&v1.SelectorConfig{
					Name:      "DefinedBeforeTest",
					Namespace: "somens",
					LabelSelector: map[string]string{
						"whendefined": "before",
					},
				},
			},
			expectedSelectorsAfterwards: []v1.SelectorConfig{
				v1.SelectorConfig{
					Name:      "DefinedBeforeTest",
					Namespace: "somens",
					LabelSelector: map[string]string{
						"whendefined": "before",
					},
				},
				v1.SelectorConfig{
					Name:      "newSelector",
					Namespace: "namespace-for-new-selector",
					LabelSelector: map[string]string{
						"whendefined": "before",
					},
				},
			},
			expectedSavedConfig: true,
		},
	}

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

	//Delete temp folder
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
		fakeConfig := &latest.Config{}
		if testCase.selectorsBefore != nil {
			fakeConfig.Dev = &v1.DevConfig{
				Selectors: testCase.selectorsBefore,
			}
		}
		configutil.SetFakeConfig(fakeConfig)
		if testCase.selectorsBefore == nil {
			fakeConfig.Dev = nil
		}

		err := AddSelector(testCase.selectorName, testCase.labelSelector, testCase.namespace, testCase.save)

		if fakeConfig.Dev == nil {
			fakeConfig.Dev = &v1.DevConfig{Selectors: []*v1.SelectorConfig{}}
		}
		assert.Equal(t, len(fakeConfig.Dev.Selectors), len(testCase.expectedSelectorsAfterwards), "Unexpected amount of selectors in testCase %s", testCase.name)
		for index, expectedSelector := range testCase.expectedSelectorsAfterwards {
			assert.Equal(t, fakeConfig.Dev.Selectors[index].Name, expectedSelector.Name, "Unexpected selector name in testCase %s", testCase.name)
			assert.Equal(t, fakeConfig.Dev.Selectors[index].Namespace, expectedSelector.Namespace, "Unexpected selector namespace in testCase %s", testCase.name)
			assert.Equal(t, len(fakeConfig.Dev.Selectors[index].LabelSelector), len(expectedSelector.LabelSelector), "Unexpected amount of labelselectors in selector %s in testCase %s", expectedSelector.Name, testCase.name)
			for key, value := range expectedSelector.LabelSelector {
				assert.Equal(t, fakeConfig.Dev.Selectors[index].LabelSelector[key], value, "Unexpected labelselector value of key %s in selector %s in testCase %s", key, expectedSelector.Name, testCase.name)
			}
		}

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		err = os.Remove(constants.DefaultConfigPath)
		assert.Equal(t, !os.IsNotExist(err), testCase.expectedSavedConfig, "Unexpectedly saved or not saved in testCase %s", testCase.name)
	}
}

type removeSelectorTestCase struct {
	name string

	removeAllFlag        bool
	selectorName         string
	labelSelector        string
	namespace            string
	defineBeforeSelector bool

	expectBeforeSelectorAfterwards bool
	expectedErr                    string
	expectedSavedConfig            bool
}

func TestRemoveSelector(t *testing.T) {
	testCases := []removeSelectorTestCase{
		removeSelectorTestCase{
			name:                           "Empty input",
			defineBeforeSelector:           true,
			expectBeforeSelectorAfterwards: true,
			expectedErr:                    "You have to specify at least one of the supported flags or specify the selectors' name",
		},
		removeSelectorTestCase{
			name:                           "Bad selector map",
			labelSelector:                  "this=does=not=work",
			defineBeforeSelector:           true,
			expectBeforeSelectorAfterwards: true,
			expectedErr:                    "Error parsing selectors: Wrong selector format: this=does=not=work",
		},
		removeSelectorTestCase{
			name:                           "Remove all",
			removeAllFlag:                  true,
			defineBeforeSelector:           true,
			expectBeforeSelectorAfterwards: false,
			expectedSavedConfig:            true,
		},
		removeSelectorTestCase{
			name:                           "Remove by name",
			selectorName:                   "DefinedBeforeTest",
			defineBeforeSelector:           true,
			expectBeforeSelectorAfterwards: false,
			expectedSavedConfig:            true,
		},
		removeSelectorTestCase{
			name:                           "Remove by labelselector",
			labelSelector:                  "whendefined=before",
			defineBeforeSelector:           true,
			expectBeforeSelectorAfterwards: false,
			expectedSavedConfig:            true,
		},
		removeSelectorTestCase{
			name:                           "Remove by namespace",
			namespace:                      "somens",
			defineBeforeSelector:           true,
			expectBeforeSelectorAfterwards: false,
			expectedSavedConfig:            true,
		},
		removeSelectorTestCase{
			name:                           "Don't remove. Wrong name",
			selectorName:                   "NoMatch",
			defineBeforeSelector:           true,
			expectBeforeSelectorAfterwards: true,
			expectedSavedConfig:            true,
		},
		removeSelectorTestCase{
			name:                           "Don't remove. Wrong labelselector",
			labelSelector:                  "match=false",
			defineBeforeSelector:           true,
			expectBeforeSelectorAfterwards: true,
			expectedSavedConfig:            true,
		},
		removeSelectorTestCase{
			name:                           "Don't remove. Wrong namespace",
			namespace:                      "wrongns",
			defineBeforeSelector:           true,
			expectBeforeSelectorAfterwards: true,
			expectedSavedConfig:            true,
		},
		removeSelectorTestCase{
			name:                           "No selectors to remove",
			removeAllFlag:                  true,
			defineBeforeSelector:           false,
			expectBeforeSelectorAfterwards: false,
			expectedSavedConfig:            false,
		},
	}

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

	//Delete temp folder
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
		fakeConfig := &latest.Config{}
		if testCase.defineBeforeSelector {
			fakeConfig.Dev = &v1.DevConfig{
				Selectors: []*v1.SelectorConfig{
					&v1.SelectorConfig{
						Name:      "DefinedBeforeTest",
						Namespace: "somens",
						LabelSelector: map[string]string{
							"whendefined": "before",
						},
					},
				},
			}
		}
		configutil.SetFakeConfig(fakeConfig)

		err := RemoveSelector(testCase.removeAllFlag, testCase.selectorName, testCase.labelSelector, testCase.namespace)

		if fakeConfig.Dev.Selectors == nil {
			fakeConfig.Dev.Selectors = []*v1.SelectorConfig{}
		}

		expectedSelectorLength := 0
		if testCase.expectBeforeSelectorAfterwards {
			expectedSelectorLength = 1
		}
		assert.Equal(t, len(fakeConfig.Dev.Selectors), expectedSelectorLength, "Unexpected amount of selectors in testCase %s", testCase.name)

		for _, selector := range fakeConfig.Dev.Selectors {
			assert.Equal(t, selector.Name, "DefinedBeforeTest", "Unexpected selector change in testCase %s", testCase.name)
			assert.Equal(t, selector.Namespace, "somens", "Unexpected selector change in testCase %s", testCase.name)
			assert.Equal(t, len(selector.LabelSelector), 1, "Unexpected selector change in testCase %s", testCase.name)
			assert.Equal(t, selector.LabelSelector["whendefined"], "before", "Unexpected selector change in testCase %s", testCase.name)
		}

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		err = os.Remove(constants.DefaultConfigPath)
		assert.Equal(t, !os.IsNotExist(err), testCase.expectedSavedConfig, "Unexpectedly saved or not saved in testCase %s", testCase.name)
	}
}
