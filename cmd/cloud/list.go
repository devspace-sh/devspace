package cloud

import (
	"strconv"

	cloudpkg "github.com/covexo/devspace/pkg/devspace/cloud"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// ListCloudCmd holds the information for the devspace add cloud commands
type ListCloudCmd struct {
	DevSpacesFlags *ListCloudDevspacesFlags
	TargetsFlags   *ListCloudTargetsFlags
}

// ListCloudDevspacesFlags holds the flag values for the devspace list cloud devspaces command
type ListCloudDevspacesFlags struct {
	Name string
}

// ListCloudTargetsFlags holds the flag values for the devspace list cloud targets command
type ListCloudTargetsFlags struct {
	DevSpaceID string
	Name       string
}

func init() {
	cmd := &ListCloudCmd{
		DevSpacesFlags: &ListCloudDevspacesFlags{},
		TargetsFlags:   &ListCloudTargetsFlags{},
	}

	listCloud := &cobra.Command{
		Use:   "list",
		Short: "List cloud provider specifics",
		Long: `
	#######################################################
	############### devspace cloud list ###################
	#######################################################
	You can list devspaces or devspace targets:
	
	* devspace cloud list devspaces
	* devspace cloud list targets
	#######################################################
	`,
		Args: cobra.NoArgs,
	}

	listCloudDevSpaces := &cobra.Command{
		Use:   "devspaces",
		Short: "Lists all user devspaces",
		Long: `
	#######################################################
	########## devspace list cloud devspaces ##############
	#######################################################
	List all cloud devspaces

	Example:
	devspace list cloud devspaces
	#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunListCloudDevspaces,
	}

	listCloudDevSpaces.Flags().StringVar(&cmd.DevSpacesFlags.Name, "name", "", "DevSpace name to show (default: all)")
	listCloud.AddCommand(listCloudDevSpaces)

	listCloudTargets := &cobra.Command{
		Use:   "targets",
		Short: "Lists all devspace targets",
		Long: `
	#######################################################
	########### devspace cloud list targets ###############
	#######################################################
	List all cloud targets

	Example:
	devspace cloud list targets
	devspace cloud list targets --name=dev
	devspace cloud list targets --devspace-id=1
	#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunListCloudTargets,
	}

	listCloudTargets.Flags().StringVar(&cmd.TargetsFlags.Name, "name", "", "Target name to show (default: all)")
	listCloudTargets.Flags().StringVar(&cmd.TargetsFlags.DevSpaceID, "devspace-id", "", "DevSpace id to use")
	listCloud.AddCommand(listCloudTargets)

	Cmd.AddCommand(listCloud)
}

// RunListCloudDevspaces executes the devspace list cloud devspaces functionality
func (cmd *ListCloudCmd) RunListCloudDevspaces(cobraCmd *cobra.Command, args []string) {
	provider, err := cloudpkg.GetCurrentProvider(log.GetInstance())
	if err != nil {
		log.Fatalf("Error getting cloud context: %v", err)
	}
	if provider == nil {
		log.Fatal("No cloud provider specified")
	}

	err = provider.PrintDevSpaces(cmd.DevSpacesFlags.Name)
	if err != nil {
		log.Fatal(err)
	}
}

// RunListCloudTargets executes the devspace list cloud targets functionality
func (cmd *ListCloudCmd) RunListCloudTargets(cobraCmd *cobra.Command, args []string) {
	provider, err := cloudpkg.GetCurrentProvider(log.GetInstance())
	if err != nil {
		log.Fatalf("Error getting cloud context: %v", err)
	}
	if provider == nil {
		log.Fatal("No cloud provider specified")
	}

	// Get generated config
	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	devSpaceID, err := strconv.Atoi(cmd.TargetsFlags.DevSpaceID)
	if err != nil {
		if generatedConfig.Cloud == nil {
			log.Fatal("No devspace id provided. Please use --devspace-id to specify the devspace id")
		}

		devSpaceID = generatedConfig.Cloud.DevSpaceID
	}

	targets, err := provider.GetDevSpaceTargetConfigs(devSpaceID)
	if err != nil {
		log.Fatalf("Error retrieving targets: %v", err)
	}

	headerColumnNames := []string{
		"Name",
		"Namespace",
		"Domain",
		"Server",
	}
	values := [][]string{}

	for _, target := range targets {
		if cmd.TargetsFlags.Name == "" || cmd.TargetsFlags.Name == target.TargetName {
			domain := ""
			if target.Domain != nil {
				domain = *target.Domain
			}

			values = append(values, []string{
				target.TargetName,
				target.Namespace,
				domain,
				target.Server,
			})
		}
	}

	if len(values) > 0 {
		log.PrintTable(headerColumnNames, values)
	} else {
		log.Info("No targets found")
	}
}
