package cmd

import (
	helmClient "github.com/covexo/devspace/pkg/devspace/clients/helm"

	"github.com/covexo/devspace/pkg/devspace/config/v1"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

type DownCmd struct {
	flags         *DownCmdFlags
	helm          *helmClient.HelmClientWrapper
	kubectl       *kubernetes.Clientset
	privateConfig *v1.PrivateConfig
	dsConfig      *v1.DevSpaceConfig
	workdir       string
}

type DownCmdFlags struct {
}

func init() {
	cmd := &DownCmd{
		flags: &DownCmdFlags{},
	}

	cobraCmd := &cobra.Command{
		Use:   "down",
		Short: "Shutdown your DevSpace",
		Long: `
#######################################################
################### devspace down #####################
#######################################################
Stops your DevSpace by removing the release via helm.
If you want to remove all DevSpace related data from
your project, use: devspace reset
#######################################################`,
		Run: cmd.Run,
	}
	rootCmd.AddCommand(cobraCmd)
}

func (cmd *DownCmd) Run(cobraCmd *cobra.Command, args []string) {

}
