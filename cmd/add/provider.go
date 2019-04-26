package add

import (
	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type providerCmd struct{}

func newProviderCmd() *cobra.Command {
	cmd := &providerCmd{}

	addProviderCmd := &cobra.Command{
		Use:   "provider",
		Short: "Adds a new cloud provider to the configuration",
		Long: `
#######################################################
############## devspace add provider ##################
#######################################################
Add a new cloud provider.

Example:
devspace add provider app.devspace.cloud
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunAddProvider,
	}

	return addProviderCmd
}

// RunAddProvider executes the "devspace add provider" functionality
func (cmd *providerCmd) RunAddProvider(cobraCmd *cobra.Command, args []string) {
	providerName := args[0]

	// Get provider configuration
	providerConfig, err := cloudpkg.LoadCloudConfig()
	if err != nil {
		log.Fatalf("Error loading provider config: %v", err)
	}

	providerConfig[providerName] = &cloudpkg.Provider{
		Name: providerName,
		Host: "https://" + providerName,
	}

	// Ensure user is logged in
	err = cloudpkg.EnsureLoggedIn(providerConfig, providerName, log.GetInstance())
	if err != nil {
		log.Fatalf("Couldn't login to provider: %v", err)
	}

	err = cloudpkg.SaveCloudConfig(providerConfig)
	if err != nil {
		log.Fatalf("Couldn't save provider config: %v", err)
	}

	log.Donef("Successfully added cloud provider %s", providerName)
}
