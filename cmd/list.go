package cmd

import (
	"os"
	"strconv"

	"github.com/covexo/devspace/pkg/devspace/config"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// ListCmd holds the information needed for the list command
type ListCmd struct {
	flags         *ListCmdFlags
	dsConfig      *v1.DevSpaceConfig
	privateConfig *v1.PrivateConfig
	appConfig     *v1.AppConfig
	workdir       string
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
		Run: cmd.RunListSync,
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
		Run: cmd.RunListPort,
	}

	listCmd.AddCommand(listPortCmd)
}

// RunListSync runs the list sync command logic
func (cmd *ListCmd) RunListSync(cobraCmd *cobra.Command, args []string) {
	loadConfig(&cmd.workdir, &cmd.privateConfig, &cmd.dsConfig)

	if len(cmd.dsConfig.SyncPaths) == 0 {
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

	syncPaths := make([][]string, 0, len(cmd.dsConfig.SyncPaths))

	// Transform values into string arrays
	for _, value := range cmd.dsConfig.SyncPaths {
		selector := ""

		for k, v := range value.LabelSelector {
			if len(selector) > 0 {
				selector += ", "
			}

			selector += k + "=" + v
		}

		excludedPaths := ""

		for _, v := range value.ExcludeRegex {
			if len(excludedPaths) > 0 {
				excludedPaths += ", "
			}

			excludedPaths += v
		}

		syncPaths = append(syncPaths, []string{
			value.ResourceType,
			selector,
			value.LocalSubPath,
			value.ContainerPath,
			excludedPaths,
		})
	}

	log.PrintTable(headerColumnNames, syncPaths)
}

// RunListPort runs the list port command logic
func (cmd *ListCmd) RunListPort(cobraCmd *cobra.Command, args []string) {
	loadConfig(&cmd.workdir, &cmd.privateConfig, &cmd.dsConfig)

	if len(cmd.dsConfig.PortForwarding) == 0 {
		log.Write("No ports are forwarded. Run `devspace add port` to add a port that should be forwarded\n")
		return
	}

	headerColumnNames := []string{
		"Type",
		"Selector",
		"Ports (Local:Remote)",
	}

	portForwards := make([][]string, 0, len(cmd.dsConfig.PortForwarding))

	// Transform values into string arrays
	for _, value := range cmd.dsConfig.PortForwarding {
		selector := ""

		for k, v := range value.LabelSelector {
			if len(selector) > 0 {
				selector += ", "
			}

			selector += k + "=" + v
		}

		portMappings := ""

		for _, v := range value.PortMappings {
			if len(portMappings) > 0 {
				portMappings += ", "
			}

			portMappings += strconv.Itoa(v.LocalPort) + ":" + strconv.Itoa(v.RemotePort)
		}

		portForwards = append(portForwards, []string{
			value.ResourceType,
			selector,
			portMappings,
		})
	}

	log.PrintTable(headerColumnNames, portForwards)
}

func loadConfig(workdir *string, privateConfig **v1.PrivateConfig, dsConfig **v1.DevSpaceConfig) {
	w, err := os.Getwd()

	if err != nil {
		log.Fatalf("Unable to determine current workdir: %s", err.Error())
	}

	workdir = &w
	*privateConfig = &v1.PrivateConfig{}
	*dsConfig = &v1.DevSpaceConfig{}

	err = config.LoadConfig(privateConfig)

	if err != nil {
		log.Fatalf("Unable to load .devspace/private.yaml: %s. Did you run `devspace init`?", err.Error())
	}

	err = config.LoadConfig(dsConfig)

	if err != nil {
		log.Fatalf("Unable to load .devspace/config.yaml: %s. Did you run `devspace init`?", err.Error())
	}
}
