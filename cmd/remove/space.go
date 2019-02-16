package remove

import (
	"strconv"

	cloudpkg "github.com/covexo/devspace/pkg/devspace/cloud"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	"github.com/covexo/devspace/pkg/util/log"
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
		Run:  cmd.RunRemoveCloudDevSpace,
	}

	spaceCmd.Flags().StringVar(&cmd.SpaceID, "id", "", "SpaceID id to use")
	spaceCmd.Flags().StringVar(&cmd.Provider, "provider", "", "Provider to use")
	spaceCmd.Flags().BoolVar(&cmd.All, "all", false, "Delete all spaces")

	return spaceCmd
}

// RunRemoveCloudDevSpace executes the devspace remove cloud devspace functionality
func (cmd *spaceCmd) RunRemoveCloudDevSpace(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}

	var provider *cloudpkg.Provider

	providerMap, err := cloudpkg.ParseCloudConfig()
	if err != nil {
		log.Fatalf("Error loading provider: %v", err)
	}

	if cmd.Provider != "" {
		provider = providerMap[cmd.Provider]
		if provider == nil {
			log.Fatalf("Couldn't find provider %s", cmd.Provider)
		}
	} else {
		// Load provider
		if configExists {
			provider, err = cloudpkg.GetCurrentProvider(log.GetInstance())
			if err != nil {
				log.Fatalf("Error getting cloud context: %v", err)
			}
		} else {
			provider = providerMap[cloudpkg.DevSpaceCloudProviderName]
			if provider == nil {
				log.Fatalf("Couldn't find provider %s", cloudpkg.DevSpaceCloudProviderName)
			}
		}
	}

	// Delete all spaces
	if cmd.All {
		spaces, err := provider.GetSpaces()
		if err != nil {
			log.Fatal(err)
		}

		for _, space := range spaces {
			err = provider.DeleteSpace(space.SpaceID)
			if err != nil {
				log.Fatal(err)
			}

			log.Donef("Deleted space %s", space.Name)
		}

		log.Done("All spaces removed")
		return
	}

	log.StartWait("Delete space")
	defer log.StopWait()

	// Get by id
	var space *generated.SpaceConfig

	if cmd.SpaceID != "" {
		spaceID, err := strconv.Atoi(cmd.SpaceID)
		if err != nil {
			log.Fatalf("Couldn't parse space id %s: %v", cmd.SpaceID, err)
		}

		space, err = provider.GetSpace(spaceID)
		if err != nil {
			log.Fatalf("Error retrieving space: %v", err)
		}
	} else if len(args) > 0 {
		space, err = provider.GetSpaceByName(args[0])
		if err != nil {
			log.Fatalf("Error retrieving space %s: %v", args[0], err)
		}
	} else {
		// Get current space
		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			log.Fatal(err)
		}

		if generatedConfig.Space == nil {
			log.Fatal("Please provide a space name or id for this command")
		}

		space = generatedConfig.Space
	}

	// Delete space remotely
	err = provider.DeleteSpace(space.SpaceID)
	if err != nil {
		log.Fatalf("Error deleting space: %v", err)
	}

	// Delete kube context
	err = cloudpkg.DeleteKubeContext(space)
	if err != nil {
		log.Fatalf("Error deleting kube context: %v", err)
	}

	if configExists {
		// Get current space
		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			log.Fatal(err)
		}

		// Remove space from generated config
		generatedConfig.Space = nil

		err = generated.SaveConfig(generatedConfig)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Donef("Deleted space %s", space.Name)
}
