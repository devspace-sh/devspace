package add

import (
	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/pkg/errors"

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
		RunE: cmd.RunAddProvider,
	}

	return addProviderCmd
}

// RunAddProvider executes the "devspace add provider" functionality
func (cmd *providerCmd) RunAddProvider(cobraCmd *cobra.Command, args []string) error {
	providerName := args[0]

	// Get provider configuration
	providerConfig, err := config.ParseProviderConfig()
	if err != nil {
		return errors.Wrap(err, "parse provider config")
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
		return errors.Wrap(err, "log into provider")
	}

	// Switch default provider to newly added provider name
	providerConfig.Default = providerName

	err = config.SaveProviderConfig(providerConfig)
	if err != nil {
		return errors.Wrap(err, "save provider config")
	}

	log.Donef("Successfully added cloud provider %s", providerName)
	return nil
}
