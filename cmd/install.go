package cmd

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/devspace-cloud/devspace/pkg/util/envutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/spf13/cobra"
)

// InstallCmd is a struct that defines a command call for "install"
type InstallCmd struct{}

// NewInstallCmd creates a new install command
func NewInstallCmd() *cobra.Command {
	cmd := &InstallCmd{}

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Installs the DevSpace CLI",
		Long: `
#######################################################
################## devspace install ###################
#######################################################
Registers the devspace executable in your PATH
variable.
#######################################################`,
		Args: cobra.NoArgs,
		Run:  cmd.Run,
	}

	return installCmd
}

// Run executes the command logic
func (cmd *InstallCmd) Run(cobraCmd *cobra.Command, args []string) {
	executablePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Unable to get executable path: %s", err)
	}

	executableDir := filepath.Dir(executablePath)
	err = envutil.AddToPath(executableDir)
	if err != nil {
		log.Fatalf("Unable to add devspace install dir to path: %s\n\nPlease add the following path manually to your PATH environment variable: %s\nSee this documentation page for help: https://devspace.cloud/docs/getting-started/installation", err, executableDir)
	}

	log.Info("DevSpace CLI has been added to your path.")

	if runtime.GOOS == "windows" {
		log.Warn("The Path variable will not be updated in already opened terminals. Please re-open the terminal if your system cannot find devspace.exe")
	}
}
