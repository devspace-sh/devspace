package remove

import (
	"github.com/covexo/devspace/pkg/devspace/configure"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type serviceCmd struct {
	RemoveAll     bool
	LabelSelector string
	Namespace     string
}

func newServiceCmd() *cobra.Command {
	cmd := &serviceCmd{}

	serviceCmd := &cobra.Command{
		Use:   "service",
		Short: "Removes one or all services from the devspace",
		Long: `
	#######################################################
	############ devspace remove image ####################
	#######################################################
	Removes one, multiple or all images from a devspace.
	If the argument is specified, the service with that name will be deleted.
	If more than one condition for deletion is specified, all services that match at least one of the conditions will be deleted.
	
	Examples:
	devspace remove service my-service
	devspace remove service --namespace=my-namespace --label-selector=environment=production,tier=frontend
	devspace remove service --all
	#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		Run:  cmd.RunRemoveService,
	}

	serviceCmd.Flags().BoolVar(&cmd.RemoveAll, "all", false, "Remove all services")
	serviceCmd.Flags().StringVar(&cmd.Namespace, "namespace", "", "Namespace of the service")
	serviceCmd.Flags().StringVar(&cmd.LabelSelector, "label-selector", "", "Label-selector of the service")

	return serviceCmd
}

// RunRemoveService executes the remove service command logic
func (cmd *serviceCmd) RunRemoveService(cobraCmd *cobra.Command, args []string) {
	var serviceName string
	if len(args) > 0 {
		serviceName = args[0]
	}

	err := configure.RemoveService(cmd.RemoveAll, serviceName, cmd.LabelSelector, cmd.Namespace)
	if err != nil {
		log.Fatal(err)
	}
}
