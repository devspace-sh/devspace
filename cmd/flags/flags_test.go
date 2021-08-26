package flags

import (
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/util/log"

	"gotest.tools/assert"
)

type useLastContextTestCase struct {
	name string

	globalFlags     GlobalFlags
	generatedConfig *generated.Config

	expectedErr string
}

func TestUseLastContext(t *testing.T) {
	testCases := []useLastContextTestCase{
		useLastContextTestCase{
			name: "Switch context to existent",
			globalFlags: GlobalFlags{
				SwitchContext: true,
			},
			generatedConfig: &generated.Config{
				ActiveProfile: "someProfile",
				Profiles: map[string]*generated.CacheConfig{
					"someProfile": &generated.CacheConfig{
						LastContext: &generated.LastContextConfig{
							Context:   "myKubeContext",
							Namespace: "myNamespace",
						},
					},
				},
			},
		},
		useLastContextTestCase{
			name:        "Nothing happens",
			globalFlags: GlobalFlags{},
		},
	}

	for _, testCase := range testCases {
		testUseLastContext(t, testCase)
	}
}

func testUseLastContext(t *testing.T, testCase useLastContextTestCase) {
	err := testCase.globalFlags.UseLastContext(testCase.generatedConfig, &log.DiscardLogger{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
}

func TestToConfigOptions(t *testing.T) {
	configOptions := (&GlobalFlags{
		Profiles:    []string{"myProfile2", "myProfile"},
		KubeContext: "myKubeContext",
		Vars:        []string{"var1", "var2"},
	}).ToConfigOptions(log.Discard)

	assert.Equal(t, configOptions.Profile, "myProfile", "ConfigOptions has wrong profile")
	assert.Equal(t, len(configOptions.ProfileParents), 1)
	assert.Equal(t, configOptions.KubeContext, "myKubeContext", "ConfigOptions has wrong kube context")
	assert.Equal(t, len(configOptions.Vars), 2, "ConfigOptions has wrong vars")
	assert.Equal(t, configOptions.Vars[0], "var1", "ConfigOptions has wrong vars")
	assert.Equal(t, configOptions.Vars[1], "var2", "ConfigOptions has wrong vars")
}
