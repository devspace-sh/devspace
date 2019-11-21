package remove

import (
	"strconv"

	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type spaceCmd struct {
	SpaceID  string
	Provider string
	All      bool
}

func newSpaceCmd() *cobra.Command {
	cmd := &spaceCmd{}

	spaceCmd := &cobra.Command{
		Use:   "space",
		Short: "Removes a cloud space",
		Long: `
#######################################################
############## devspace remove space ##################
#######################################################
Removes a cloud space.

Example:
devspace remove space myspace
devspace remove space --id=1
devspace remove space --all
#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		RunE: cmd.RunRemoveCloudDevSpace,
	}

	spaceCmd.Flags().StringVar(&cmd.SpaceID, "id", "", "SpaceID id to use")
	spaceCmd.Flags().StringVar(&cmd.Provider, "provider", "", "Cloud Provider to use")
	spaceCmd.Flags().BoolVar(&cmd.All, "all", false, "Delete all spaces")

	return spaceCmd
}

// RunRemoveCloudDevSpace executes the devspace remove cloud devspace functionality
func (cmd *spaceCmd) RunRemoveCloudDevSpace(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot(log.GetInstance())
	if err != nil {
		return err
	}

	// Get provider
	provider, err := cloudpkg.GetProvider(cmd.Provider, log.GetInstance())
	if err != nil {
		return errors.Wrap(err, "get provider")
	}

	// Delete all spaces
	if cmd.All {
		spaces, err := provider.GetSpaces()
		if err != nil {
			return err
		}

		for _, space := range spaces {
			err = provider.DeleteSpace(space)
			if err != nil {
				return err
			}

			err = cloudpkg.DeleteKubeContext(space)
			if err != nil {
				return errors.Wrap(err, "delete kube context")
			}

			log.Donef("Deleted space %s", space.Name)
		}

		log.Done("All spaces removed")
		return nil
	}

	log.StartWait("Delete space")
	defer log.StopWait()

	// Get by id
	var space *latest.Space

	if cmd.SpaceID != "" {
		spaceID, err := strconv.Atoi(cmd.SpaceID)
		if err != nil {
			return errors.Wrap(err, "parse space id")
		}

		space, err = provider.GetSpace(spaceID)
		if err != nil {
			return errors.Wrap(err, "get space")
		}
	} else if len(args) > 0 {
		space, err = provider.GetSpaceByName(args[0])
		if err != nil {
			return errors.Wrap(err, "get space")
		}
	} else {
		return errors.New("Please provide a space name or id for this command")
	}

	// Delete space remotely
	err = provider.DeleteSpace(space)
	if err != nil {
		return errors.Wrap(err, "delete space")
	}

	// Delete kube context
	err = cloudpkg.DeleteKubeContext(space)
	if err != nil {
		return errors.Wrap(err, "delete kube context")
	}

	if configExists {
		// Get current space
		generatedConfig, err := generated.LoadConfig("")
		if err != nil {
			return err
		}

		if generatedConfig.GetActive().LastContext != nil && generatedConfig.GetActive().LastContext.Context != "" {
			spaceID, _, err := kubeconfig.GetSpaceID(generatedConfig.GetActive().LastContext.Context)
			if err == nil && spaceID == space.SpaceID {
				// Remove cached namespace from generated config if it belongs to the space that is being deleted
				generatedConfig.GetActive().LastContext = nil
			}
		}

		err = generated.SaveConfig(generatedConfig)
		if err != nil {
			return err
		}
	}

	log.Donef("Deleted space %s", space.Name)
	return nil
}
