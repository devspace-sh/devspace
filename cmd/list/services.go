package list

import (
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type servicesCmd struct{}

func newServicesCmd() *cobra.Command {
	cmd := &servicesCmd{}

	servicesCmd := &cobra.Command{
		Use:   "services",
		Short: "Lists all services",
		Long: `
	#######################################################
	############## devspace list services #################
	#######################################################
	Lists the services that are defined in the DevSpace
	#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunListService,
	}

	return servicesCmd
}

// RunListService runs the list service command logic
func (cmd *servicesCmd) RunListService(cobraCmd *cobra.Command, args []string) {
	config := configutil.GetConfig()

	if config.DevSpace.Services == nil || len(*config.DevSpace.Services) == 0 {
		log.Info("No services are configured. Run `devspace add service` to add new service\n")
		return
	}

	headerColumnNames := []string{
		"Name",
		"Namespace",
		"Type",
		"Selector",
		"Container",
	}

	services := make([][]string, 0, len(*config.DevSpace.Services))

	// Transform values into string arrays
	for _, value := range *config.DevSpace.Services {
		selector := ""
		for k, v := range *value.LabelSelector {
			if len(selector) > 0 {
				selector += ", "
			}

			selector += k + "=" + *v
		}

		resourceType := "pod"
		if value.ResourceType != nil {
			resourceType = *value.ResourceType
		}

		// TODO: should we skip this error?
		namespace, _ := configutil.GetDefaultNamespace(config)
		if value.Namespace != nil {
			namespace = *value.Namespace
		}

		containerName := ""
		if value.ContainerName != nil {
			containerName = *value.ContainerName
		}

		services = append(services, []string{
			*value.Name,
			namespace,
			resourceType,
			selector,
			containerName,
		})
	}

	log.PrintTable(headerColumnNames, services)
}
