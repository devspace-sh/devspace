package add

import (
	"github.com/covexo/devspace/pkg/devspace/configure"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type deploymentCmd struct {
	Namespace string
	Manifests string
	Chart     string
}

func newDeploymentCmd() *cobra.Command {
	cmd := &deploymentCmd{}

	addDeploymentCmd := &cobra.Command{
		Use:   "deployment",
		Short: "Add a deployment",
		Long: ` 
	#######################################################
	############# devspace add deployment #################
	#######################################################
	Add a new deployment (kubernetes manifests or 
	helm chart) to your devspace, that will be deployed
	
	Examples:
	devspace add deployment my-deployment --chart=chart/
	devspace add deployment my-deployment --manifests=kube/pod.yaml
	devspace add deployment my-deployment --manifests=kube/* --namespace=devspace
	#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunAddDeployment,
	}

	addDeploymentCmd.Flags().StringVar(&cmd.Namespace, "namespace", "", "The namespace to use for deploying")
	addDeploymentCmd.Flags().StringVar(&cmd.Manifests, "manifests", "", "The kubernetes manifests to deploy (glob pattern are allowed, comma separated)")
	addDeploymentCmd.Flags().StringVar(&cmd.Chart, "chart", "", "The helm chart to deploy")

	return addDeploymentCmd
}

// RunAddDeployment executes the add deployment command logic
func (cmd *deploymentCmd) RunAddDeployment(cobraCmd *cobra.Command, args []string) {
	err := configure.AddDeployment(args[0], cmd.Namespace, cmd.Manifests, cmd.Chart)
	if err != nil {
		log.Fatal(err)
	}

	log.Donef("Successfully added %s as new deployment", args[0])
}
