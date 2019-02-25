package remove

import (
	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type contextCmd struct {
	All bool
}

func newContextCmd() *cobra.Command {
	cmd := &contextCmd{}

	contextCmd := &cobra.Command{
		Use:   "context",
		Short: "Removes a cloud space kubectl context",
		Long: `
#######################################################
############# devspace remove context #################
#######################################################
Removes a cloud space kubectl context

Example:
devspace remove context myspace
devspace remove context --all
#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		Run:  cmd.RunRemoveContext,
	}

	contextCmd.Flags().BoolVar(&cmd.All, "all", false, "Delete all kubectl contexts created from spaces")

	return contextCmd
}

// RunRemoveContext executes the devspace remove context functionality
func (cmd *contextCmd) RunRemoveContext(cobraCmd *cobra.Command, args []string) {
	provider, err := cloudpkg.GetCurrentProvider(log.GetInstance())
	if err != nil {
		log.Fatalf("Error getting cloud context: %v", err)
	}

	// Delete all spaces
	if cmd.All {
		spaces, err := provider.GetSpaces()
		if err != nil {
			log.Fatal(err)
		}

		for _, space := range spaces {
			// Delete kube context
			err = cloudpkg.DeleteKubeContext(space)
			if err != nil {
				log.Fatalf("Error deleting kube context: %v", err)
			}

			log.Donef("Deleted kubectl context for space %s", space.Name)
		}

		log.Done("All space kubectl contexts removed")
		return
	} else if len(args) == 0 {
		log.Fatal("Please specify a space name or the --all flag")
	}

	// Retrieve space
	space, err := provider.GetSpaceByName(args[0])
	if err != nil {
		log.Fatalf("Error retrieving space %s: %v", args[0], err)
	}

	// Delete kube context
	err = cloudpkg.DeleteKubeContext(space)
	if err != nil {
		log.Fatalf("Error deleting kube context: %v", err)
	}

	log.Donef("Kubectl context deleted for space %s", args[0])
}
