package list

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/chart"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type availableComponentsCmd struct{}

func newAvailableComponentsCmd() *cobra.Command {
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
		RunE: cmd.RunListAvailableComponents,
	}

	return availableComponentsCmd
}

// RunListPackage runs the list available components logic
func (cmd *availableComponentsCmd) RunListAvailableComponents(cobraCmd *cobra.Command, args []string) error {
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

	log.PrintTable(log.GetInstance(), headerColumnNames, values)
	return nil
}
