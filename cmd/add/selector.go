package add

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/configure"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type selectorCmd struct {
	LabelSelector string
	Namespace     string
}

func newSelectorCmd() *cobra.Command {
	cmd := &selectorCmd{}

	selectorCmd := &cobra.Command{
		Use:   "selector",
		Short: "Add a selector",
		Long: ` 
#######################################################
############# devspace add selector ###################
#######################################################
Add a new selector to your DevSpace configuration

Examples:
devspace add selector my-selector --namespace=my-namespace
devspace add selector my-selector --label-selector=environment=production,tier=frontend
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunAddSelector,
	}

	selectorCmd.Flags().StringVar(&cmd.Namespace, "namespace", "", "The namespace of the selector")
	selectorCmd.Flags().StringVar(&cmd.LabelSelector, "label-selector", "", "The label-selector of the selector")

	return selectorCmd
}

// RunAddSelector executes the add selector command logic
func (cmd *selectorCmd) RunAddSelector(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	err = configure.AddSelector(args[0], cmd.LabelSelector, cmd.Namespace, true)
	if err != nil {
		log.Fatal(err)
	}

	log.Donef("Successfully added new service %v", args[0])
}
