package use

import (
	"github.com/covexo/devspace/pkg/devspace/cloud"
	cloudpkg "github.com/covexo/devspace/pkg/devspace/cloud"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type spaceCmd struct {
	context bool
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

	useSpace.Flags().BoolVar(&cmd.context, "context", true, "Create/Update kubectl context for space")

	return useSpace
}

// RunUseDevSpace executes the functionality devspace cloud use devspace
func (cmd *spaceCmd) RunUseSpace(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find any devspace configuration. Please run `devspace init`")
	}

	// Erase currently used space
	if args[0] == "none" {
		// Get generated config
		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			log.Fatal(err)
		}

		generatedConfig.Space = nil

		err = generated.SaveConfig(generatedConfig)
		if err != nil {
			log.Fatal(err)
		}

		log.Info("Successfully erased space")
		return
	}

	// Get cloud provider from config
	provider, err := cloudpkg.GetCurrentProvider(log.GetInstance())
	if err != nil {
		log.Fatalf("Error getting cloud context: %v", err)
	}
	if provider == nil {
		log.Fatal("No cloud provider specified")
	}

	log.StartWait("Retrieving space")

	spaceConfig, err := provider.GetSpaceByName(args[0])
	if err != nil {
		log.Fatalf("Error retrieving devspaces: %v", err)
	}

	log.StopWait()

	// Get generated config
	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	generatedConfig.Space = spaceConfig

	err = generated.SaveConfig(generatedConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Update kube config
	if cmd.context {
		err = cloud.UpdateKubeConfig(cloud.GetKubeContextNameFromSpace(spaceConfig), spaceConfig, true)
		if err != nil {
			log.Fatalf("Error saving kube config: %v", err)
		}
	}

	log.Donef("Successfully configured config to use space %s", spaceConfig.Name)
}
