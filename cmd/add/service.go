package add

import (
	"github.com/covexo/devspace/pkg/devspace/configure"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type serviceCmd struct {
	LabelSelector string
	Namespace     string
}

func newServiceCmd() *cobra.Command {
	cmd := &serviceCmd{}

	addServiceCmd := &cobra.Command{
		Use:   "service",
		Short: "Add a service",
		Long: ` 
	#######################################################
	############# devspace add service ####################
	#######################################################
	Add a new service to your devspace
	
	Examples:
	devspace add service my-service --namespace=my-namespace
	devspace add service my-service --label-selector=environment=production,tier=frontend
	#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunAddService,
	}

	addServiceCmd.Flags().StringVar(&cmd.Namespace, "namespace", "", "The namespace of the service")
	addServiceCmd.Flags().StringVar(&cmd.LabelSelector, "label-selector", "", "The label-selector of the service")

	return addServiceCmd
}

// RunAddService executes the add image command logic
func (cmd *serviceCmd) RunAddService(cobraCmd *cobra.Command, args []string) {
	err := configure.AddService(args[0], cmd.LabelSelector, cmd.Namespace)
	if err != nil {
		log.Fatal(err)
	}

	log.Donef("Successfully added new service %v", args[0])
}
