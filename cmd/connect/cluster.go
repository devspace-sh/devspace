package connect

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

type clusterCmd struct {
	Provider string

	UseHostNetwork bool
	Options        *cloud.ConnectClusterOptions
}

func newClusterCmd(f factory.Factory) *cobra.Command {
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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunConnectCluster(f, cobraCmd, args)
		}}

	clusterCmd.Flags().StringVar(&cmd.Provider, "provider", "", "The cloud provider to use")

	clusterCmd.Flags().BoolVar(&cmd.Options.DeployAdmissionController, "admission-controller", true, "Deploy the admission controller")
	clusterCmd.Flags().BoolVar(&cmd.Options.DeployGatekeeper, "gatekeeper", true, "Deploy the gatekeeper")
	clusterCmd.Flags().BoolVar(&cmd.Options.DeployGatekeeperRules, "gatekeeper-rules", true, "Deploy the gatekeeper default rules")
	clusterCmd.Flags().BoolVar(&cmd.Options.DeployIngressController, "ingress-controller", true, "Deploy an ingress controller")
	clusterCmd.Flags().BoolVar(&cmd.UseHostNetwork, "use-hostnetwork", false, "Use the host network for the ingress controller instead of a loadbalancer")
	clusterCmd.Flags().BoolVar(&cmd.Options.DeployCertManager, "cert-manager", true, "Deploy a cert manager")
	clusterCmd.Flags().BoolVar(&cmd.Options.Public, "public", false, "Connects a new public cluster")
	clusterCmd.Flags().StringVar(&cmd.Options.KubeContext, "context", "", "The kube context to use")
	clusterCmd.Flags().StringVar(&cmd.Options.Key, "key", "", "The encryption key to use")
	clusterCmd.Flags().StringVar(&cmd.Options.ClusterName, "name", "", "The cluster name to create")

	clusterCmd.Flags().BoolVar(&cmd.Options.OpenUI, "open-ui", false, "Opens the UI and displays the cluster overview")
	clusterCmd.Flags().BoolVar(&cmd.Options.UseDomain, "use-domain", false, "Use an automatic domain for the cluster")
	clusterCmd.Flags().StringVar(&cmd.Options.Domain, "domain", "", "The domain to use")

	return clusterCmd
}

// RunConnectCluster executes the connect cluster command logic
func (cmd *clusterCmd) RunConnectCluster(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	log := f.GetLog()
	// Get provider
	provider, err := f.GetProvider(cmd.Provider, log)
	if err != nil {
		return err
	}

	// Check if use host network was used
	if cobraCmd.Flags().Changed("use-hostnetwork") {
		cmd.Options.UseHostNetwork = &cmd.UseHostNetwork
	}

	// Connect cluster
	err = provider.ConnectCluster(cmd.Options)
	if err != nil {
		return err
	}

	log.Donef("Successfully connected cluster to DevSpace Cloud. \n\nYou can now run:\n- `%s` to create a new space\n- `%s` to list all connected clusters", ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace ui", "white+b"), ansi.Color("devspace list clusters", "white+b"))
	return nil
}
