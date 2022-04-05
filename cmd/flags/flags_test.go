package flags

import (
	"testing"

	"gotest.tools/assert"
)

func TestToConfigOptions(t *testing.T) {
	configOptions := (&GlobalFlags{
		Profiles:    []string{"myProfile2", "myProfile"},
		KubeContext: "myKubeContext",
		Vars:        []string{"var1", "var2"},
	}).ToConfigOptions()

	assert.Equal(t, configOptions.Profiles[0], "myProfile2", "ConfigOptions has wrong profiles")
	assert.Equal(t, configOptions.Profiles[1], "myProfile", "ConfigOptions has wrong profiles")
	assert.Equal(t, len(configOptions.Vars), 2, "ConfigOptions has wrong vars")
	assert.Equal(t, configOptions.Vars[0], "var1", "ConfigOptions has wrong vars")
	assert.Equal(t, configOptions.Vars[1], "var2", "ConfigOptions has wrong vars")
}
