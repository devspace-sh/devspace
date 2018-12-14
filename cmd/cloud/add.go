package cloud

import (
	"net/url"
	"strings"

	cloudpkg "github.com/covexo/devspace/pkg/devspace/cloud"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// AddCloudCmd holds the information for the devspace add cloud commands
type AddCloudCmd struct {
	ProviderFlags *AddCloudProviderFlags
}

// AddCloudProviderFlags holds the flag values for the devspace add cloud provider command
type AddCloudProviderFlags struct {
	Name string
}

func init() {
	cmd := &AddCloudCmd{
		ProviderFlags: &AddCloudProviderFlags{},
	}

	addCloud := &cobra.Command{
		Use:   "add",
		Short: "Add cloud provider specifics",
		Long: `
	#######################################################
	################ devspace cloud add ###################
	#######################################################
	You can add a cloud provider with:
	
	* devspace cloud add provider 
	#######################################################
	`,
		Args: cobra.NoArgs,
	}

	addCloudProvider := &cobra.Command{
		Use:   "provider",
		Short: "Adds a new cloud provider to the configuration",
		Long: `
	#######################################################
	########### devspace cloud add provider ###############
	#######################################################
	Add a new cloud provider.

	Example:
	devspace cloud add provider http://cli.devspace-cloud.com
	#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunAddCloudProvider,
	}

	addCloudProvider.Flags().StringVar(&cmd.ProviderFlags.Name, "name", "", "Cloud provider name to use")
	addCloud.AddCommand(addCloudProvider)

	Cmd.AddCommand(addCloud)
}

// RunAddCloudProvider executes the devspace add cloud provider functionality
func (cmd *AddCloudCmd) RunAddCloudProvider(cobraCmd *cobra.Command, args []string) {
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
	providerConfig, err := cloudpkg.ParseCloudConfig()
	if err != nil {
		log.Fatalf("Error loading provider config: %v", err)
	}

	providerConfig[providerName] = &cloudpkg.Provider{
		Name: providerName,
		Host: args[0],
	}

	err = cloudpkg.SaveCloudConfig(providerConfig)
	if err != nil {
		log.Fatalf("Couldn't save provider config: %v", err)
	}

	log.Donef("Successfully added cloud provider %s", providerName)
}
