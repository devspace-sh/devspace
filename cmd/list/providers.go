package list

import (
	"strconv"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type providersCmd struct{}

func newProvidersCmd() *cobra.Command {
	cmd := &providersCmd{}

	providersCmd := &cobra.Command{
		Use:   "providers",
		Short: "Lists all providers",
		Long: `
#######################################################
############# devspace list providers #################
#######################################################
Lists the providers that exist
#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunListProviders,
	}

	return providersCmd
}

// RunListProviders runs the list providers command logic
func (cmd *providersCmd) RunListProviders(cobraCmd *cobra.Command, args []string) {
	// Get provider configuration
	providerConfig, err := config.ParseProviderConfig()
	if err != nil {
		log.Fatalf("Error loading provider config: %v", err)
	}

	headerColumnNames := []string{
		"Name",
		"IsDefault",
		"Host",
		"Is logged in",
	}

	providerRows := make([][]string, 0, len(providerConfig.Providers))

	// Transform values into string arrays
	for _, provider := range providerConfig.Providers {
		providerRows = append(providerRows, []string{
			provider.Name,
			strconv.FormatBool(provider.Name == providerConfig.Default),
			provider.Host,
			strconv.FormatBool(provider.Key != ""),
		})
	}

	log.PrintTable(log.GetInstance(), headerColumnNames, providerRows)
}
