package list

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/chart"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type availableComponentsCmd struct{}

func newAvailableComponentsCmd(f factory.Factory) *cobra.Command {
	cmd := &availableComponentsCmd{}

	availableComponentsCmd := &cobra.Command{
		Use:   "available-components",
		Short: "Lists all available components",
		Long: `
#######################################################
######### devspace list available-components ##########
#######################################################
Lists all the available components that can be used
in devspace
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunListAvailableComponents(f, cobraCmd, args)
		},
	}

	return availableComponentsCmd
}

// RunListPackage runs the list available components logic
func (cmd *availableComponentsCmd) RunListAvailableComponents(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	logger := f.GetLog()
	headerColumnNames := []string{
		"Name",
		"Description",
	}
	values := [][]string{}

	components, err := chart.ListAvailableComponents()
	if err != nil {
		return errors.Wrap(err, "list components")
	}

	for _, component := range components {
		values = append(values, []string{
			component.Name,
			component.Description,
		})
	}

	log.PrintTable(logger, headerColumnNames, values)
	return nil
}
