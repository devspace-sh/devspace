package add

import (
	"net/url"
	"strings"

	cloudpkg "github.com/covexo/devspace/pkg/devspace/cloud"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type providerCmd struct {
	Name string
}

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
devspace add provider https://app.devspace.cloud
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunAddProvider,
	}

	addProviderCmd.Flags().StringVar(&cmd.Name, "name", "", "Cloud provider name to use")

	return addProviderCmd
}

// RunAddProvider executes the devspace add provider functionality
func (cmd *providerCmd) RunAddProvider(cobraCmd *cobra.Command, args []string) {
	providerName := cmd.Name
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
