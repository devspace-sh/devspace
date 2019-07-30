package remove

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type providerCmd struct {
	Name string
}

func newProviderCmd() *cobra.Command {
	cmd := &providerCmd{}

	providerCmd := &cobra.Command{
		Use:   "provider",
		Short: "Removes a cloud provider from the configuration",
		Long: `
#######################################################
############ devspace remove provider #################
#######################################################
Removes a cloud provider.

Example:
devspace remove provider app.devspace.cloud
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunRemoveCloudProvider,
	}

	providerCmd.Flags().StringVar(&cmd.Name, "name", "", "Cloud provider name to use")

	return providerCmd
}

// RunRemoveCloudProvider executes the devspace remove cloud provider functionality
func (cmd *providerCmd) RunRemoveCloudProvider(cobraCmd *cobra.Command, args []string) {
	providerName := args[0]

	// Get provider configuration
	providerConfig, err := config.ParseProviderConfig()
	if err != nil {
		log.Fatalf("Error loading provider config: %v", err)
	}
	if config.GetProvider(providerConfig, providerName) == nil {
		log.Failf("Couldn't find cloud provider %s", providerName)
	}

	newProviders := make([]*latest.Provider, 0, len(providerConfig.Providers)-1)
	for _, provider := range providerConfig.Providers {
		if provider.Name == providerName {
			continue
		}

		newProviders = append(newProviders, provider)
	}

	providerConfig.Providers = newProviders

	// Change default provider if necessary
	if providerConfig.Default == providerName {
		providerConfig.Default = config.DevSpaceCloudProviderName
	}

	err = config.SaveProviderConfig(providerConfig)
	if err != nil {
		log.Fatalf("Couldn't save provider config: %v", err)
	}

	log.Donef("Successfully removed cloud provider %s", providerName)
}
