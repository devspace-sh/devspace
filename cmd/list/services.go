package list

import (
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type selectorsCmd struct{}

func newSelectorsCmd() *cobra.Command {
	cmd := &selectorsCmd{}

	selectorsCmd := &cobra.Command{
		Use:   "selectors",
		Short: "Lists all selectors",
		Long: `
	#######################################################
	############# devspace list selectors #################
	#######################################################
	Lists the selectors that are defined in the DevSpace
	#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunListService,
	}

	return selectorsCmd
}

// RunListService runs the list service command logic
func (cmd *selectorsCmd) RunListService(cobraCmd *cobra.Command, args []string) {
	config := configutil.GetConfig()

	if config.DevSpace.Selectors == nil || len(*config.DevSpace.Selectors) == 0 {
		log.Info("No selectors are configured. Run `devspace add selector` to add new selector\n")
		return
	}

	headerColumnNames := []string{
		"Name",
		"Namespace",
		"Type",
		"Label Selector",
		"Container",
	}

	selectors := make([][]string, 0, len(*config.DevSpace.Selectors))

	// Transform values into string arrays
	for _, value := range *config.DevSpace.Selectors {
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

		selectors = append(selectors, []string{
			*value.Name,
			namespace,
			resourceType,
			selector,
			containerName,
		})
	}

	log.PrintTable(headerColumnNames, selectors)
}
