package list

import (
	"strconv"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type portsCmd struct {
	*flags.GlobalFlags
}

func newPortsCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunListPort(f, cobraCmd, args)
		}}

	return portsCmd
}

// RunListPort runs the list port command logic
func (cmd *portsCmd) RunListPort(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	logger := f.GetLog()
	// Set config root
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(logger)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	configInterface, err := configLoader.Load(cmd.ToConfigOptions(), logger)
	if err != nil {
		return err
	}

	config := configInterface.Config()
	if config.Dev.Ports == nil || len(config.Dev.Ports) == 0 {
		logger.Info("No ports are forwarded. Run `devspace add port` to add a port that should be forwarded\n")
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

	log.PrintTable(logger, headerColumnNames, portForwards)
	return nil
}
