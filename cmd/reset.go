package cmd

import (
	helmClient "github.com/covexo/devspace/pkg/devspace/clients/helm"

	"github.com/covexo/devspace/pkg/devspace/config/v1"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

type ResetCmd struct {
	flags         *ResetCmdFlags
	helm          *helmClient.HelmClientWrapper
	kubectl       *kubernetes.Clientset
	privateConfig *v1.PrivateConfig
	dsConfig      *v1.DevSpaceConfig
	workdir       string
}

type ResetCmdFlags struct {
}

func init() {
	cmd := &ResetCmd{
		flags: &ResetCmdFlags{},
	}

	cobraCmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset your project",
		Long: `
#######################################################
################### devspace reset ####################
#######################################################
Resets your project by removing all DevSpace related 
data from your project and your cluster, including:
1. DevSpace release (cluster)
2. Docker registry (cluster)
3. DevSpace config files in .devspace/ (local)

Use the flag --all-data to also remove:
1. Tiller server (cluster)
2. Helm home (local)

If you simply want to shutdown your DevSpace, use the 
command: devspace down
#######################################################`,
		Run: cmd.Run,
	}
	rootCmd.AddCommand(cobraCmd)
}

func (cmd *ResetCmd) Run(cobraCmd *cobra.Command, args []string) {

}
