package remove

import (
	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
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
	providerConfig, err := cloudpkg.LoadCloudConfig()
	if err != nil {
		log.Fatalf("Error loading provider config: %v", err)
	}

	if _, ok := providerConfig[providerName]; ok == false {
		log.Failf("Couldn't find cloud provider %s", providerName)
	}

	delete(providerConfig, providerName)

	err = cloudpkg.SaveCloudConfig(providerConfig)
	if err != nil {
		log.Fatalf("Couldn't save provider config: %v", err)
	}

	log.Donef("Successfully removed cloud provider %s", providerName)
}
