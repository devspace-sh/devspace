package cmd

import (
	"strconv"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// ListCmd holds the information needed for the list command
type ListCmd struct {
	flags   *ListCmdFlags
	workdir string
}

// ListCmdFlags holds the possible flags for the list command
type ListCmdFlags struct {
}

func init() {
	cmd := &ListCmd{
		flags: &ListCmdFlags{},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Lists configuration",
		Long: `
	#######################################################
	#################### devspace list ####################
	#######################################################
	Lists the following configurations:
	
	* Sync paths (sync)
	* Forwarded ports (port)
	#######################################################
	`,
		Args: cobra.NoArgs,
	}

	rootCmd.AddCommand(listCmd)

	listSyncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Lists sync configuration",
		Long: `
	#######################################################
	################# devspace list sync ##################
	#######################################################
	Lists the sync configuration
	#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunListSync,
	}

	listCmd.AddCommand(listSyncCmd)

	listPortCmd := &cobra.Command{
		Use:   "port",
		Short: "Lists port forwarding configuration",
		Long: `
	#######################################################
	################ devspace list port ###################
	#######################################################
	Lists the port forwarding configuration
	#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunListPort,
	}

	listCmd.AddCommand(listPortCmd)
}

// RunListSync runs the list sync command logic
func (cmd *ListCmd) RunListSync(cobraCmd *cobra.Command, args []string) {
	config := configutil.GetConfig(false)

	if len(*config.DevSpace.Sync) == 0 {
		log.Write("No sync paths are configured. Run `devspace add sync` to add new sync path\n")
		return
	}

	headerColumnNames := []string{
		"Type",
		"Selector",
		"Local Path",
		"Container Path",
		"Excluded Paths",
	}

	syncPaths := make([][]string, 0, len(*config.DevSpace.Sync))

	// Transform values into string arrays
	for _, value := range *config.DevSpace.Sync {
		selector := ""

		for k, v := range *value.LabelSelector {
			if len(selector) > 0 {
				selector += ", "
			}

			selector += k + "=" + *v
		}

		excludedPaths := ""

		if value.ExcludePaths != nil {
			for _, v := range *value.ExcludePaths {
				if len(excludedPaths) > 0 {
					excludedPaths += ", "
				}

				excludedPaths += v
			}
		}
		syncPaths = append(syncPaths, []string{
			*value.ResourceType,
			selector,
			*value.LocalSubPath,
			*value.ContainerPath,
			excludedPaths,
		})
	}

	log.PrintTable(headerColumnNames, syncPaths)
}

// RunListPort runs the list port command logic
func (cmd *ListCmd) RunListPort(cobraCmd *cobra.Command, args []string) {
	config := configutil.GetConfig(false)

	if len(*config.DevSpace.PortForwarding) == 0 {
		log.Write("No ports are forwarded. Run `devspace add port` to add a port that should be forwarded\n")
		return
	}

	headerColumnNames := []string{
		"Type",
		"Selector",
		"Ports (Local:Remote)",
	}

	portForwards := make([][]string, 0, len(*config.DevSpace.PortForwarding))

	// Transform values into string arrays
	for _, value := range *config.DevSpace.PortForwarding {
		selector := ""

		for k, v := range *value.LabelSelector {
			if len(selector) > 0 {
				selector += ", "
			}

			selector += k + "=" + *v
		}

		portMappings := ""

		for _, v := range *value.PortMappings {
			if len(portMappings) > 0 {
				portMappings += ", "
			}

			portMappings += strconv.Itoa(*v.LocalPort) + ":" + strconv.Itoa(*v.RemotePort)
		}

		portForwards = append(portForwards, []string{
			*value.ResourceType,
			selector,
			portMappings,
		})
	}

	log.PrintTable(headerColumnNames, portForwards)
}
