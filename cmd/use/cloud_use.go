package cloud

import (
	"strconv"

	cloudpkg "github.com/covexo/devspace/pkg/devspace/cloud"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// UseCmd holds the information for the devspace cloud use commands
type UseCmd struct {
	DevSpaceFlags *UseDevSpaceFlags
}

// UseDevSpaceFlags holds the flag values for the devspace cloud use devspace command
type UseDevSpaceFlags struct {
	ID string
}

func init() {
	cmd := &UseCmd{
		DevSpaceFlags: &UseDevSpaceFlags{},
	}

	useCmd := &cobra.Command{
		Use:   "use",
		Short: "",
		Long: `
	#######################################################
	################ devspace cloud use ###################
	#######################################################
	You can use an existing devspace for your project with:
	
	* devspace cloud use devspace 
	#######################################################
	`,
		Args: cobra.NoArgs,
	}

	useDevSpace := &cobra.Command{
		Use:   "devspace",
		Short: "Use an existing devspace for a project",
		Long: `
	#######################################################
	########### devspace cloud use devspace ###############
	#######################################################
	Add a new cloud provider.

	Example:
	devspace cloud use devspace my-devspace
	devspace cloud use devspace --id=1
	#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		Run:  cmd.RunUseDevSpace,
	}

	useDevSpace.Flags().StringVar(&cmd.DevSpaceFlags.ID, "id", "", "DevSpace id to use")
	useCmd.AddCommand(useDevSpace)

	Cmd.AddCommand(useCmd)
}

// RunUseDevSpace executes the functionality devspace cloud use devspace
func (cmd *UseCmd) RunUseDevSpace(cobraCmd *cobra.Command, args []string) {
	if cmd.DevSpaceFlags.ID != "" && len(args) > 0 {
		log.Fatalf("Please only specify either --id or name")
	}

	// Get cloud provider from config
	provider, err := cloudpkg.GetCurrentProvider(log.GetInstance())
	if err != nil {
		log.Fatalf("Error getting cloud context: %v", err)
	}
	if provider == nil {
		log.Fatal("No cloud provider specified")
	}

	log.StartWait("Retrieving devspaces")

	devspaces, err := provider.GetDevSpaces()
	if err != nil {
		log.Fatalf("Error retrieving devspaces: %v", err)
	}

	log.StopWait()

	var devSpaceConfig *cloudpkg.DevSpaceConfig

	if len(args) > 0 {
		devSpaceName := args[0]
		foundDevSpaces := []*cloudpkg.DevSpaceConfig{}

		for _, devSpace := range devspaces {
			if devSpace.Name == devSpaceName {
				foundDevSpaces = append(foundDevSpaces, devSpace)
			}
		}

		if len(foundDevSpaces) == 1 {
			devSpaceConfig = foundDevSpaces[0]
		} else if len(foundDevSpaces) == 0 {
			log.Errorf("No DevSpace with name %s found. Please choose one of:", devSpaceName)
			err = provider.PrintDevSpaces("")
			if err != nil {
				log.Fatal(err)
			}

			return
		} else {
			log.Errorf("Multiple devspaces with name %s found. Please use the --id flag and use one of:", devSpaceName)
			err = provider.PrintDevSpaces(devSpaceName)
			if err != nil {
				log.Fatal(err)
			}

			return
		}
	} else if cmd.DevSpaceFlags.ID != "" {
		devSpaceID, err := strconv.Atoi(cmd.DevSpaceFlags.ID)
		if err != nil {
			log.Fatalf("Error parsing --id: %v", err)
		}

		for _, devSpace := range devspaces {
			if devSpace.DevSpaceID == devSpaceID {
				devSpaceConfig = devSpace
			}
		}

		if devSpaceConfig == nil {
			log.Fatalf("DevSpace with id %d not found", devSpaceID)
		}
	}

	// Get generated config
	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	generatedConfig.Cloud = &generated.CloudConfig{
		DevSpaceID:   devSpaceConfig.DevSpaceID,
		Name:         devSpaceConfig.Name,
		ProviderName: provider.Name,
	}

	err = generated.SaveConfig(generatedConfig)
	if err != nil {
		log.Fatal(err)
	}

	log.Donef("Successfully configured project to use devspace %s", devSpaceConfig.Name)
}
