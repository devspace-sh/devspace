package list

import (
	"strconv"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type portsCmd struct{}

func newPortsCmd() *cobra.Command {
	cmd := &portsCmd{}

	portsCmd := &cobra.Command{
		Use:   "ports",
		Short: "Lists port forwarding configurations",
		Long: `
	#######################################################
	############### devspace list ports ###################
	#######################################################
	Lists the port forwarding configurations
	#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunListPort,
	}

	return portsCmd
}

// RunListPort runs the list port command logic
func (cmd *portsCmd) RunListPort(cobraCmd *cobra.Command, args []string) {
	config := configutil.GetConfig()

	if config.DevSpace.Ports == nil || len(*config.DevSpace.Ports) == 0 {
		log.Info("No ports are forwarded. Run `devspace add port` to add a port that should be forwarded\n")
		return
	}

	headerColumnNames := []string{
		"Service",
		"Type",
		"Selector",
		"Ports (Local:Remote)",
	}

	portForwards := make([][]string, 0, len(*config.DevSpace.Ports))

	// Transform values into string arrays
	for _, value := range *config.DevSpace.Ports {
		service := ""
		selector := ""

		if value.Selector != nil {
			service = *value.Selector
		} else {
			for k, v := range *value.LabelSelector {
				if len(selector) > 0 {
					selector += ", "
				}

				selector += k + "=" + *v
			}
		}

		portMappings := ""
		for _, v := range *value.PortMappings {
			if len(portMappings) > 0 {
				portMappings += ", "
			}

			portMappings += strconv.Itoa(*v.LocalPort) + ":" + strconv.Itoa(*v.RemotePort)
		}

		resourceType := "pod"
		if value.ResourceType != nil {
			resourceType = *value.ResourceType
		}

		portForwards = append(portForwards, []string{
			service,
			resourceType,
			selector,
			portMappings,
		})
	}

	log.PrintTable(headerColumnNames, portForwards)
}
