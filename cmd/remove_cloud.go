package cmd

import (
	"net/url"
	"strings"

	"github.com/covexo/devspace/pkg/devspace/cloud"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

var removeCloud = &cobra.Command{
	Use:   "cloud",
	Short: "Remove cloud provider specifics",
	Long: `
#######################################################
############### devspace remove cloud #################
#######################################################
You can remove a cloud provider with:

* devspace remove cloud provider 
#######################################################
`,
	Args: cobra.NoArgs,
}

// RemoveCloudCmd holds the information for the devspace remove cloud commands
type RemoveCloudCmd struct {
	ProviderFlags *RemoveCloudProviderFlags
}

// RemoveCloudProviderFlags holds the flag values for the devspace remove cloud provider command
type RemoveCloudProviderFlags struct {
	Name string
}

func init() {
	cmd := &RemoveCloudCmd{
		ProviderFlags: &RemoveCloudProviderFlags{},
	}

	removeCloudProvider := &cobra.Command{
		Use:   "provider",
		Short: "Removes a cloud provider from the configuration",
		Long: `
	#######################################################
	######### devspace remove cloud provider ##############
	#######################################################
	Removes a cloud provider.

	Example:
	devspace remove cloud provider http://cli.devspace-cloud.com
	#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunRemoveCloudProvider,
	}

	removeCloudProvider.Flags().StringVar(&cmd.ProviderFlags.Name, "name", "", "Cloud provider name to use")
	removeCloud.AddCommand(removeCloudProvider)
}

// RunRemoveCloudProvider executes the devspace remove cloud provider functionality
func (cmd *RemoveCloudCmd) RunRemoveCloudProvider(cobraCmd *cobra.Command, args []string) {
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

	if _, ok := providerConfig[providerName]; ok == false {
		log.Failf("Couldn't find cloud provider %s", providerName)
	}

	delete(providerConfig, providerName)

	err = cloud.SaveCloudConfig(providerConfig)
	if err != nil {
		log.Fatalf("Couldn't save provider config: %v", err)
	}

	log.Donef("Successfully removed cloud provider %s", providerName)
}
