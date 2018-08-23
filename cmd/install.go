package cmd

import (
	"os"
	"path/filepath"

	"github.com/covexo/devspace/pkg/util/envutil"

	helmClient "github.com/covexo/devspace/pkg/devspace/clients/helm"

	"github.com/covexo/devspace/pkg/devspace/config/v1"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

type InstallCmd struct {
	flags         *InstallCmdFlags
	helm          *helmClient.HelmClientWrapper
	kubectl       *kubernetes.Clientset
	privateConfig *v1.PrivateConfig
	dsConfig      *v1.DevSpaceConfig
	workdir       string
}

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
		Run: cmd.Run,
	}
	rootCmd.AddCommand(cobraCmd)
}

func (cmd *InstallCmd) Run(cobraCmd *cobra.Command, args []string) {
	executablePath, err := os.Executable()

	if err != nil {
		panic(err)
	}
	executableDir := filepath.Dir(executablePath)
	err = envutil.AddToPath(executableDir)

	if err != nil {
		log.WithError(err).Panic("Unable to add devspace install dir to path.")
	}
}
