package list

import (
	"strconv"

	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type providersCmd struct{}

func newProvidersCmd(f factory.Factory) *cobra.Command {
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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunListProviders(f, cobraCmd, args)
		}}

	return providersCmd
}

// RunListProviders runs the list providers command logic
func (cmd *providersCmd) RunListProviders(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	logger := f.GetLog()
	// Get provider configuration
	loader := f.NewCloudConfigLoader()
	providerConfig, err := loader.Load()
	if err != nil {
		return errors.Wrap(err, "log into provider")
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

	log.PrintTable(logger, headerColumnNames, providerRows)
	return nil
}
