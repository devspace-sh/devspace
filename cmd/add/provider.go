package add

import (
	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"

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
	providerConfig, err := config.ParseProviderConfig()
	if err != nil {
		log.Fatalf("Error loading provider config: %v", err)
	}
	if providerConfig.Providers == nil {
		providerConfig.Providers = []*latest.Provider{}
	}

	provider := config.GetProvider(providerConfig, providerName)
	if provider == nil {
		providerConfig.Providers = append(providerConfig.Providers, &latest.Provider{
			Name: providerName,
			Host: "https://" + providerName,
		})
	} else {
		provider.Host = "https://" + providerName
	}

	// Ensure user is logged in
	err = cloudpkg.EnsureLoggedIn(providerConfig, providerName, log.GetInstance())
	if err != nil {
		log.Fatalf("Couldn't login to provider: %v", err)
	}

	// Switch default provider to newly added provider name
	providerConfig.Default = providerName

	err = config.SaveProviderConfig(providerConfig)
	if err != nil {
		log.Fatalf("Couldn't save provider config: %v", err)
	}

	log.Donef("Successfully added cloud provider %s", providerName)
}
