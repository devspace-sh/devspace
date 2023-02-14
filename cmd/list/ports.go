package list

import (
	"context"

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
	configLoader, _ := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(logger)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	configInterface, err := configLoader.Load(context.TODO(), nil, cmd.ToConfigOptions(), logger)
	if err != nil {
		return err
	}

	config := configInterface.Config()
	portForwards := make([][]string, 0)
	for _, dev := range config.Dev {
		if dev.Ports == nil || len(dev.Ports) == 0 {
			continue
		}
		selector := ""
		for k, v := range dev.LabelSelector {
			if len(selector) > 0 {
				selector += ", "
			}
			selector += k + "=" + v
		}
		// Transform values into string arrays
		for _, value := range dev.Ports {
			portForwards = append(portForwards, []string{
				dev.ImageSelector,
				selector,
				value.Port,
			})
		}
	}
	if len(portForwards) == 0 {
		logger.Info("No ports are forwarded.\n")
		return nil
	}
	headerColumnNames := []string{
		"ImageSelector",
		"LabelSelector",
		"Ports (Local:Remote)",
	}
	log.PrintTable(logger, headerColumnNames, portForwards)
	return nil
}
