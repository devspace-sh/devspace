package cmd

import (
	"net/url"
	"strings"

	"github.com/covexo/devspace/pkg/devspace/cloud"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

var addCloud = &cobra.Command{
	Use:   "cloud",
	Short: "Add cloud provider specifics",
	Long: `
#######################################################
################ devspace add cloud ###################
#######################################################
You can add a cloud provider with:

* devspace add cloud provider 
#######################################################
`,
	Args: cobra.NoArgs,
}

// AddCloudCmd holds the information for the devspace add cloud commands
type AddCloudCmd struct {
	ProviderFlags *AddCloudProviderFlags
}

// AddCloudProviderFlags holds the flag values for the devspace add cloud provider command
type AddCloudProviderFlags struct {
	Name string
}

func init() {
	cmd := &AddCloudCmd{
		ProviderFlags: &AddCloudProviderFlags{},
	}

	addCloudProvider := &cobra.Command{
		Use:   "provider",
		Short: "Adds a new cloud provider to the configuration",
		Long: `
	#######################################################
	########### devspace add cloud provider ###############
	#######################################################
	Add a new cloud provider.

	Example:
	devspace add cloud provider http://cli.devspace-cloud.com
	#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunAddCloudProvider,
	}

	addCloudProvider.Flags().StringVar(&cmd.ProviderFlags.Name, "name", "", "Cloud provider name to use")
	addCloud.AddCommand(addCloudProvider)
}

// RunAddCloudProvider executes the devspace add cloud provider functionality
func (cmd *AddCloudCmd) RunAddCloudProvider(cobraCmd *cobra.Command, args []string) {
	providerName := cmd.ProviderFlags.Name
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
	providerConfig, err := cloud.ParseCloudConfig()
	if err != nil {
		log.Fatalf("Error loading provider config: %v", err)
	}

	providerConfig[providerName] = &cloud.Provider{
		Name: providerName,
		Host: args[0],
	}

	err = cloud.SaveCloudConfig(providerConfig)
	if err != nil {
		log.Fatalf("Couldn't save provider config: %v", err)
	}

	log.Donef("Successfully added cloud provider %s", providerName)
}
