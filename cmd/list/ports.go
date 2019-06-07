package list

import (
	"strconv"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
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
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	config := configutil.GetConfig()

	if config.Dev.Ports == nil || len(*config.Dev.Ports) == 0 {
		log.Info("No ports are forwarded. Run `devspace add port` to add a port that should be forwarded\n")
		return
	}

	headerColumnNames := []string{
		"Selector",
		"LabelSelector",
		"Ports (Local:Remote)",
	}

	portForwards := make([][]string, 0, len(*config.Dev.Ports))

	// Transform values into string arrays
	for _, value := range *config.Dev.Ports {
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
		if value.PortMappings != nil {
			for _, v := range *value.PortMappings {
				if len(portMappings) > 0 {
					portMappings += ", "
				}

				portMappings += strconv.Itoa(*v.LocalPort) + ":" + strconv.Itoa(*v.RemotePort)
			}
		}

		portForwards = append(portForwards, []string{
			service,
			selector,
			portMappings,
		})
	}

	log.PrintTable(log.GetInstance(), headerColumnNames, portForwards)
}
