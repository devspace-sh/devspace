package flags

import (
	"os"
	"testing"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/env"
	"github.com/spf13/cobra"
	"gotest.tools/assert"
)

func Test_ApplyExtraFlagsWithEmpty(t *testing.T) {
	mycmd := cobra.Command{}
	flags, err := ApplyExtraFlags(&mycmd, []string{}, false)
	assert.NilError(t, err, "Error applying extra flags")
	assert.Equal(t, 0, len(flags), "Flags should be empty")
}

func Test_ApplyExtraFlagsWithDevspaceFlags(t *testing.T) {
	devspaceFlags := env.GlobalGetEnv("DEVSPACE_FLAGS")
	assert.Equal(t, "", devspaceFlags, "DEVSPACE_FLAGS should be empty")

	devspaceFlags = "-s --kubeconfig /path/to/kubeconfig --debug"
	err := os.Setenv("DEVSPACE_FLAGS", devspaceFlags)
	assert.NilError(t, err, "Error setting DEVSPACE_FLAGS")
	mycmd := &cobra.Command{}
	persistentFlags := mycmd.PersistentFlags()
	_ = flags.SetGlobalFlags(persistentFlags)

	flags, err := ApplyExtraFlags(mycmd, []string{}, false)
	assert.NilError(t, err, "Error applying extra flags")
	assert.Equal(t, 4, len(flags), "Flags should have 4 elements")

	kubeconfig := env.GlobalGetEnv("KUBECONFIG")
	assert.Equal(t, "/path/to/kubeconfig", kubeconfig, "Path should be /path/to/kubeconfig")
}
