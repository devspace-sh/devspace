package remove

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ProviderCmd struct {
	Name string
}

func newProviderCmd(f factory.Factory) *cobra.Command {
	cmd := &ProviderCmd{}

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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunRemoveCloudProvider(f, cobraCmd, args)
		}}

	providerCmd.Flags().StringVar(&cmd.Name, "name", "", "Cloud provider name to use")

	return providerCmd
}

// RunRemoveCloudProvider executes the devspace remove cloud provider functionality
func (cmd *ProviderCmd) RunRemoveCloudProvider(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	providerName := args[0]
	log := f.GetLog()

	// Get provider configuration
	loader := f.NewCloudConfigLoader()
	providerConfig, err := loader.Load()
	if err != nil {
		return errors.Wrap(err, "parse provider config")
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

	err = loader.Save(providerConfig)
	if err != nil {
		return errors.Wrap(err, "save provider config")
	}

	log.Donef("Successfully removed cloud provider %s", providerName)
	return nil
}
