package cmd

import (
	"os"
	"path/filepath"

	"github.com/covexo/devspace/pkg/util/envutil"
	"github.com/covexo/devspace/pkg/util/log"

	helmClient "github.com/covexo/devspace/pkg/devspace/clients/helm"

	"github.com/covexo/devspace/pkg/devspace/config/v1"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

// InstallCmd is a struct that defines a command call for "install"
type InstallCmd struct {
	flags    *InstallCmdFlags
	helm     *helmClient.HelmClientWrapper
	kubectl  *kubernetes.Clientset
	dsConfig *v1.DevSpaceConfig
	workdir  string
}

// InstallCmdFlags are the flags available for the install-command
type InstallCmdFlags struct {
}

func init() {
	cmd := &InstallCmd{
		flags: &InstallCmdFlags{},
	}

	cobraCmd := &cobra.Command{
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
	rootCmd.AddCommand(cobraCmd)
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
		log.Fatalf("Unable to add devspace install dir to path: %s", err)
	}
}
