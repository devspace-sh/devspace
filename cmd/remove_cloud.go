package cmd

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/covexo/devspace/pkg/devspace/cloud"
	"github.com/covexo/devspace/pkg/devspace/config/generated"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

var removeCloud = &cobra.Command{
	Use:   "cloud",
	Short: "Remove cloud provider specifics",
	Long: `
#######################################################
############### devspace remove cloud #################
#######################################################
You can remove a cloud provider with:

* devspace remove cloud provider 
#######################################################
`,
	Args: cobra.NoArgs,
}

// RemoveCloudCmd holds the information for the devspace remove cloud commands
type RemoveCloudCmd struct {
	ProviderFlags *RemoveCloudProviderFlags
	DevSpaceFlags *RemoveCloudDevSpaceFlags
}

// RemoveCloudProviderFlags holds the flag values for the devspace remove cloud provider command
type RemoveCloudProviderFlags struct {
	Name string
}

// RemoveCloudDevSpaceFlags holds the flag values for the devspace remove cloud devspace command
type RemoveCloudDevSpaceFlags struct {
	DevSpaceID string
	All        bool
}

func init() {
	cmd := &RemoveCloudCmd{
		ProviderFlags: &RemoveCloudProviderFlags{},
		DevSpaceFlags: &RemoveCloudDevSpaceFlags{},
	}

	removeCloudProvider := &cobra.Command{
		Use:   "provider",
		Short: "Removes a cloud provider from the configuration",
		Long: `
	#######################################################
	######### devspace remove cloud provider ##############
	#######################################################
	Removes a cloud provider.

	Example:
	devspace remove cloud provider http://cli.devspace-cloud.com
	#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunRemoveCloudProvider,
	}

	removeCloudProvider.Flags().StringVar(&cmd.ProviderFlags.Name, "name", "", "Cloud provider name to use")
	removeCloud.AddCommand(removeCloudProvider)

	removeCloudDevSpace := &cobra.Command{
		Use:   "devspace",
		Short: "Removes a cloud devspace",
		Long: `
	#######################################################
	######### devspace remove cloud devspace ##############
	#######################################################
	Removes a cloud devspace.

	Example:
	devspace remove cloud devspace
	devspace remove cloud devspace --devspace-id=1
	devspace remove cloud devspace --all
	#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunRemoveCloudDevSpace,
	}

	removeCloudDevSpace.Flags().StringVar(&cmd.DevSpaceFlags.DevSpaceID, "devspace-id", "", "DevSpace id to use")
	removeCloudDevSpace.Flags().BoolVar(&cmd.DevSpaceFlags.All, "all", false, "Delete all devspaces")
	removeCloud.AddCommand(removeCloudDevSpace)
}

// RunRemoveCloudDevSpace executes the devspace remove cloud devspace functionality
func (cmd *RemoveCloudCmd) RunRemoveCloudDevSpace(cobraCmd *cobra.Command, args []string) {
	provider, err := cloud.GetCurrentProvider(log.GetInstance())
	if err != nil {
		log.Fatalf("Error getting cloud context: %v", err)
	}
	if provider == nil {
		log.Fatal("No cloud provider specified")
	}

	if cmd.DevSpaceFlags.All {
		devSpaces, err := provider.GetDevSpaces()
		if err != nil {
			log.Fatal(err)
		}

		for _, devSpace := range devSpaces {
			err = provider.DeleteDevSpace(devSpace.DevSpaceID)
			if err != nil {
				log.Fatal(err)
			}

			log.Donef("Deleted devspace %s", devSpace.Name)
		}

		log.Done("All devspaces removed")
		return
	}

	// Get generated config
	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	devSpaceID, err := strconv.Atoi(cmd.DevSpaceFlags.DevSpaceID)
	if err != nil {
		if generatedConfig.Cloud == nil {
			log.Fatal("No devspace id provided. Please use --devspace-id to specify the devspace id")
		}

		devSpaceID = generatedConfig.Cloud.DevSpaceID
	}

	err = provider.DeleteDevSpace(devSpaceID)
	if err != nil {
		log.Fatalf("Error deleting devspace: %v", err)
	}

	log.Donef("Deleted devspace %d", devSpaceID)
}

// RunRemoveCloudProvider executes the devspace remove cloud provider functionality
func (cmd *RemoveCloudCmd) RunRemoveCloudProvider(cobraCmd *cobra.Command, args []string) {
	providerName := cmd.ProviderFlags.Name
	if providerName == "" {
		u, err := url.Parse(args[0])
		if err != nil {
			log.Fatal(err)
		}

		parts := strings.Split(u.Hostname(), ".")
		if len(parts) >= 2 {
			providerName = parts[len(parts)-2] + "." + parts[len(parts)-1]
		} else {
			providerName = u.Hostname()
		}
	}

	// Get provider configuration
	providerConfig, err := cloud.ParseCloudConfig()
	if err != nil {
		log.Fatalf("Error loading provider config: %v", err)
	}

	if _, ok := providerConfig[providerName]; ok == false {
		log.Failf("Couldn't find cloud provider %s", providerName)
	}

	delete(providerConfig, providerName)

	err = cloud.SaveCloudConfig(providerConfig)
	if err != nil {
		log.Fatalf("Couldn't save provider config: %v", err)
	}

	log.Donef("Successfully removed cloud provider %s", providerName)
}
