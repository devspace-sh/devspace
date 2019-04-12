package use

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type spaceCmd struct {
	provider string
}

func newSpaceCmd() *cobra.Command {
	cmd := &spaceCmd{}

	useSpace := &cobra.Command{
		Use:   "space",
		Short: "Use an existing space for the current configuration",
		Long: `
#######################################################
################ devspace use space ###################
#######################################################
Use an existing space for the current configuration

Example:
devspace use space my-space
devspace use space none    // stop using a space
#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		Run:  cmd.RunUseSpace,
	}

	useSpace.Flags().StringVar(&cmd.provider, "provider", "", "The cloud provider to use")

	return useSpace
}

// RunUseDevSpace executes the functionality "devspace use space"
func (cmd *spaceCmd) RunUseSpace(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}

	// Erase currently used space
	if args[0] == "none" {
		// Set tiller env
		err = cloudpkg.SetTillerNamespace(nil)
		if err != nil {
			log.Warnf("Couldn't set tiller namespace environment variable: %v", err)
		}

		if !configExists {
			return
		}

		// Get generated config
		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			log.Fatal(err)
		}

		generatedConfig.CloudSpace = nil

		err = generated.SaveConfig(generatedConfig)
		if err != nil {
			log.Fatal(err)
		}

		log.Info("Successfully erased space from config")
		return
	}

	// Check if user has specified a certain provider
	var cloudProvider *string
	if cmd.provider != "" {
		cloudProvider = &cmd.provider
	}

	// Get cloud provider from config
	provider, err := cloudpkg.GetProvider(cloudProvider, log.GetInstance())
	if err != nil {
		log.Fatalf("Error getting cloud context: %v", err)
	}
	if provider == nil {
		log.Fatal("No cloud provider specified")
	}

	log.StartWait("Retrieving Space details")

	space, err := provider.GetSpaceByName(args[0])
	if err != nil {
		log.Fatalf("Error retrieving Spaces details: %v", err)
	}

	log.StopWait()

	// Change kube context
	kubeContext := cloud.GetKubeContextNameFromSpace(space.Name, space.ProviderName)

	// Get service account
	serviceAccount, err := provider.GetServiceAccount(space)
	if err != nil {
		log.Fatalf("Error retrieving space service account: %v", err)
	}

	err = cloud.UpdateKubeConfig(kubeContext, serviceAccount, true)
	if err != nil {
		log.Fatalf("Error saving kube config: %v", err)
	}

	// Set tiller env
	err = cloudpkg.SetTillerNamespace(serviceAccount)
	if err != nil {
		log.Warnf("Couldn't set tiller namespace environment variable: %v", err)
	}

	if configExists {
		// Get generated config
		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			log.Fatal(err)
		}

		generatedConfig.CloudSpace = &generated.CloudSpaceConfig{
			SpaceID:      space.SpaceID,
			ProviderName: space.ProviderName,
			Name:         space.Name,
			Owner:        space.Owner.Name,
			OwnerID:      space.Owner.OwnerID,
			KubeContext:  kubeContext,
			Created:      space.Created,
			Domain:       space.Domain,
		}

		err = generated.SaveConfig(generatedConfig)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Donef("Successfully configured config to use space %s", space.Name)
}
