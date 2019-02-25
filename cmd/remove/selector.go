package remove

import (
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/configure"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type selectorCmd struct {
	RemoveAll     bool
	LabelSelector string
	Namespace     string
}

func newSelectorCmd() *cobra.Command {
	cmd := &selectorCmd{}

	selectorCmd := &cobra.Command{
		Use:   "selector",
		Short: "Removes one or all selectors from the devspace",
		Long: `
#######################################################
############ devspace remove selector #################
#######################################################
Removes one, multiple or all selectors from a devspace.
If the argument is specified, the selector with that name will be deleted.
If more than one condition for deletion is specified, all selectors that match at least one of the conditions will be deleted.

Examples:
devspace remove selector my-selector
devspace remove selector --namespace=my-namespace --label-selector=environment=production,tier=frontend
devspace remove selector --all
#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		Run:  cmd.RunRemoveSelector,
	}

	selectorCmd.Flags().BoolVar(&cmd.RemoveAll, "all", false, "Remove all selectors")
	selectorCmd.Flags().StringVar(&cmd.Namespace, "namespace", "", "Namespace of the selector")
	selectorCmd.Flags().StringVar(&cmd.LabelSelector, "label-selector", "", "Label-selector of the selector")

	return selectorCmd
}

// RunRemoveSelector executes the remove service command logic
func (cmd *selectorCmd) RunRemoveSelector(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find any devspace configuration. Please run `devspace init`")
	}

	var serviceName string
	if len(args) > 0 {
		serviceName = args[0]
	}

	err = configure.RemoveSelector(cmd.RemoveAll, serviceName, cmd.LabelSelector, cmd.Namespace)
	if err != nil {
		log.Fatal(err)
	}
}
