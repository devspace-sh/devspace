package use

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type providerCmd struct{}

func newProviderCmd() *cobra.Command {
	cmd := &providerCmd{}

	return &cobra.Command{
		Use:   "provider",
		Short: "Change the default provider",
		Long: `
#######################################################
############### devspace use provider #################
#######################################################
Use a specific cloud provider as default for future
commands

Example:
devspace use provider my.domain.com
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunUseProvider,
	}
}

// RunUseProvider executes the "devspace use provider" command logic
func (*providerCmd) RunUseProvider(cobraCmd *cobra.Command, args []string) {
	providerName := args[0]

	// Get provider configuration
	providerConfig, err := config.ParseProviderConfig()
	if err != nil {
		log.Fatalf("Error loading provider config: %v", err)
	}

	provider := config.GetProvider(providerConfig, providerName)
	if provider == nil {
		log.Fatalf("Error provider %s does not exist! Did you run `devspace add provider %s` first?", providerName, providerName)
	}

	providerConfig.Default = provider.Name
	err = config.SaveProviderConfig(providerConfig)
	if err != nil {
		log.Fatalf("Couldn't save provider config: %v", err)
	}

	log.Donef("Successfully changed default cloud provider to %s", providerName)
}
