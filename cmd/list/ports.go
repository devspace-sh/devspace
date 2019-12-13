package list

import (
	"strconv"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type portsCmd struct {
	*flags.GlobalFlags
}

func newPortsCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &portsCmd{GlobalFlags: globalFlags}

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
		RunE: cmd.RunListPort,
	}

	return portsCmd
}

// RunListPort runs the list port command logic
func (cmd *portsCmd) RunListPort(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configLoader := loader.NewConfigLoader(cmd.ToConfigOptions(), log.GetInstance())
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	config, err := configLoader.Load()
	if err != nil {
		return err
	}

	if config.Dev.Ports == nil || len(config.Dev.Ports) == 0 {
		log.GetInstance().Info("No ports are forwarded. Run `devspace add port` to add a port that should be forwarded\n")
		return nil
	}

	headerColumnNames := []string{
		"Image",
		"LabelSelector",
		"Ports (Local:Remote)",
	}

	portForwards := make([][]string, 0, len(config.Dev.Ports))

	// Transform values into string arrays
	for _, value := range config.Dev.Ports {
		selector := ""
		for k, v := range value.LabelSelector {
			if len(selector) > 0 {
				selector += ", "
			}

			selector += k + "=" + v
		}

		portMappings := ""
		if value.PortMappings != nil {
			for _, v := range value.PortMappings {
				if len(portMappings) > 0 {
					portMappings += ", "
				}

				remotePort := *v.LocalPort
				if v.RemotePort != nil {
					remotePort = *v.RemotePort
				}

				portMappings += strconv.Itoa(*v.LocalPort) + ":" + strconv.Itoa(remotePort)
			}
		}

		portForwards = append(portForwards, []string{
			value.ImageName,
			selector,
			portMappings,
		})
	}

	log.PrintTable(log.GetInstance(), headerColumnNames, portForwards)
	return nil
}
