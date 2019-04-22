package connect

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

type clusterCmd struct {
	Provider string

	UseHostNetwork bool
	Options        *cloud.ConnectClusterOptions
}

func newClusterCmd() *cobra.Command {
	cmd := &clusterCmd{
		Options: &cloud.ConnectClusterOptions{},
	}

	clusterCmd := &cobra.Command{
		Use:   "cluster",
		Short: "Connects an existing cluster to DevSpace Cloud",
		Long: `
#######################################################
############ devspace connect cluster #################
#######################################################
Connects an existing cluster to DevSpace Cloud.

Examples:
devspace connect cluster 
#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunConnectCluster,
	}

	clusterCmd.Flags().StringVar(&cmd.Provider, "provider", "", "The cloud provider to use")

	clusterCmd.Flags().BoolVar(&cmd.Options.DeployAdmissionController, "admission-controller", true, "Deploy the admission controller")
	clusterCmd.Flags().BoolVar(&cmd.Options.DeployIngressController, "ingress-controller", true, "Deploy an ingress controller")
	clusterCmd.Flags().BoolVar(&cmd.UseHostNetwork, "use-hostnetwork", false, "Use the host netowkr for the ingress controller instead of a loadbalancer")
	clusterCmd.Flags().BoolVar(&cmd.Options.DeployCertManager, "cert-manager", true, "Deploy a cert manager")
	clusterCmd.Flags().StringVar(&cmd.Options.KubeContext, "context", "", "The kube context to use")
	clusterCmd.Flags().StringVar(&cmd.Options.Key, "key", "", "The encryption key to use")
	clusterCmd.Flags().StringVar(&cmd.Options.ClusterName, "name", "", "The cluster name to create")

	clusterCmd.Flags().BoolVar(&cmd.Options.UseDomain, "use-domain", true, "Use an automatic domain for the cluster")
	clusterCmd.Flags().StringVar(&cmd.Options.Domain, "domain", "", "The domain to use")

	return clusterCmd
}

// RunConnectCluster executes the connect cluster command logic
func (cmd *clusterCmd) RunConnectCluster(cobraCmd *cobra.Command, args []string) {
	// Check if user has specified a certain provider
	var cloudProvider *string
	if cmd.Provider != "" {
		cloudProvider = &cmd.Provider
	}

	// Get provider
	provider, err := cloud.GetProvider(cloudProvider, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	// Check if use host network was used
	if cobraCmd.Flags().Changed("use-hostnetwork") {
		cmd.Options.UseHostNetwork = &cmd.UseHostNetwork
	}

	// Connect cluster
	err = provider.ConnectCluster(cmd.Options)
	if err != nil {
		log.Fatal(err)
	}

	log.Donef("Successfully connected cluster to DevSpace Cloud. You can now run:\n- `%s` to create a new space\n- `%s` to open the ui and configure cluster access and users\n- `%s` to list all connected clusters", ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace ui", "white+b"), ansi.Color("devspace list clusters", "white+b"))
}
