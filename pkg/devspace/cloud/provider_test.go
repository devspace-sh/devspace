package cloud

import (
	"testing"

	testconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/testing"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	log "github.com/devspace-cloud/devspace/pkg/util/log/testing"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type getCloudProviderWithOptionsTestCase struct {
	name string

	useProviderName string
	key             string
	relogin         bool
	config          *latest.Config
	answers         []string

	expectedErr      string
	expectedProvider *latest.Provider
}

func TestGetProviderWithOptions(t *testing.T) {
	testCases := []getCloudProviderWithOptionsTestCase{
		getCloudProviderWithOptionsTestCase{
			name:            "invalid providername",
			useProviderName: "notThere",
			config: &latest.Config{
				Providers: []*latest.Provider{
					&latest.Provider{Name: "prov1"},
					&latest.Provider{Name: "prov2"},
				},
			},
			expectedErr: "Cloud provider not found! Did you run `devspace add provider [url]`? Existing cloud providers: prov1 prov2 ",
		},
		getCloudProviderWithOptionsTestCase{
			name: "default providername",
			config: &latest.Config{
				Default: "prov1",
				Providers: []*latest.Provider{
					&latest.Provider{Name: "prov1", Key: "someKey"},
					&latest.Provider{Name: "prov2"},
				},
			},
			expectedProvider: &latest.Provider{Name: "prov1", Key: "someKey"},
		},
		getCloudProviderWithOptionsTestCase{
			name: "providername from answer",
			config: &latest.Config{
				Providers: []*latest.Provider{
					&latest.Provider{Name: "prov1"},
					&latest.Provider{Name: "prov2", Key: "someKey"},
				},
			},
			answers:          []string{"prov2"},
			expectedProvider: &latest.Provider{Name: "prov2", Key: "someKey"},
		},
	}

	for _, testCase := range testCases {
		testGetProviderWithOptions(t, testCase)
	}
}

func testGetProviderWithOptions(t *testing.T, testCase getCloudProviderWithOptionsTestCase) {
	loader := testconfig.NewLoader(testCase.config)

	logger := log.NewFakeLogger()
	for _, answer := range testCase.answers {
		logger.Survey.SetNextAnswer(answer)
	}

	provider, err := GetProviderWithOptions(testCase.useProviderName, testCase.key, testCase.relogin, loader, logger)

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Error in testCase %s", testCase.name)
		provAsYaml, _ := yaml.Marshal(provider.GetConfig())
		expectedAsYaml, _ := yaml.Marshal(testCase.expectedProvider)
		assert.Equal(t, string(provAsYaml), string(expectedAsYaml), "Unexpected provider in testCase %s.\nExpected:\n%s\nActual:\n%s", testCase.name, string(expectedAsYaml), string(provAsYaml))
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
	}

}

type saveTestCase struct {
	name string

	provider provider
	config   *latest.Config

	expectedConfig *latest.Config
	expectedErr    string
}

func TestSave(t *testing.T) {
	testCases := []saveTestCase{
		saveTestCase{
			name: "Save new provider",
			provider: provider{
				Provider: latest.Provider{
					Name: "newProv",
					Host: "hostOfNewProv",
				},
			},
			config: &latest.Config{},
			expectedConfig: &latest.Config{
				Providers: []*latest.Provider{
					&latest.Provider{
						Name: "newProv",
						Host: "hostOfNewProv",
					},
				},
			},
		},
		saveTestCase{
			name: "Update provider",
			provider: provider{
				Provider: latest.Provider{
					Name: "existingProv2",
					Host: "newHost",
				},
			},
			config: &latest.Config{
				Providers: []*latest.Provider{
					&latest.Provider{
						Name: "existingProv1",
						Host: "oldHost",
					},
					&latest.Provider{
						Name: "existingProv2",
						Host: "oldHost",
					},
				},
			},
			expectedConfig: &latest.Config{
				Providers: []*latest.Provider{
					&latest.Provider{
						Name: "existingProv1",
						Host: "oldHost",
					},
					&latest.Provider{
						Name: "existingProv2",
						Host: "newHost",
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		testSave(t, testCase)
	}
}

func testSave(t *testing.T, testCase saveTestCase) {
	testCase.provider.loader = testconfig.NewLoader(testCase.config)

	err := testCase.provider.Save()

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Error in testCase %s", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
	}

	config, _ := testCase.provider.loader.Load()
	configAsYaml, _ := yaml.Marshal(config)
	expectedAsYaml, _ := yaml.Marshal(testCase.expectedConfig)
	assert.Equal(t, string(configAsYaml), string(expectedAsYaml), "Unexpected config in testCase %s.\nExpected:\n%s\nActual config:\n%s", testCase.name, string(expectedAsYaml), string(configAsYaml))
}
