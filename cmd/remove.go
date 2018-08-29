package cmd

import (
	"strconv"
	"strings"

	"github.com/covexo/devspace/pkg/devspace/config"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// RemoveCmd holds the information needed for the remove command
type RemoveCmd struct {
	syncFlags     *removeSyncCmdFlags
	portFlags     *removePortCmdFlags
	dsConfig      *v1.DevSpaceConfig
	privateConfig *v1.PrivateConfig
	workdir       string
}

type removeSyncCmdFlags struct {
	Selector      string
	LocalPath     string
	ContainerPath string
	RemoveAll     bool
}

type removePortCmdFlags struct {
	Selector  string
	Ports     string
	RemoveAll bool
}

func init() {
	cmd := &RemoveCmd{
		syncFlags: &removeSyncCmdFlags{},
		portFlags: &removePortCmdFlags{},
	}

	removeCmd := &cobra.Command{
		Use:   "remove",
		Short: "Changes devspace configuration",
		Long: `
	#######################################################
	################## devspace remove ####################
	#######################################################
	You can remove the following configuration with the
	remove command:
	
	* Sync paths (sync)
	* Forwarded ports (port)
	#######################################################
	`,
	}

	rootCmd.AddCommand(removeCmd)

	removeSyncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Remove sync paths from the devspace",
		Long: `
	#######################################################
	############### devspace remove sync ##################
	#######################################################
	Remove sync paths from the devspace

	How to use:
	devspace remove sync --local=app
	devspace remove sync --container=/app
	devspace remove sync --selector=release=test
	devspace remove sync --all
	#######################################################
	`,
		Run: cmd.RunRemoveSync,
	}

	removeCmd.AddCommand(removeSyncCmd)

	removeSyncCmd.Flags().StringVar(&cmd.syncFlags.Selector, "selector", "", "Comma separated key=value selector list (e.g. release=test)")
	removeSyncCmd.Flags().StringVar(&cmd.syncFlags.LocalPath, "local", "", "Relative local path to remove")
	removeSyncCmd.Flags().StringVar(&cmd.syncFlags.ContainerPath, "container", "", "Absolute container path to remove")
	removeSyncCmd.Flags().BoolVar(&cmd.syncFlags.RemoveAll, "all", false, "Remove all configured sync paths")

	removePortCmd := &cobra.Command{
		Use:   "port",
		Short: "Removes forwarded ports from a devspace",
		Long: `
	#######################################################
	############### devspace remove port ##################
	#######################################################
	Removes port mappings from the devspace configuration:
	devspace remove port --ports 8080,3000
	devspace remove port --selector=release=test
	devspace remove port --all
	#######################################################
	`,
		Run: cmd.RunRemovePort,
	}

	removePortCmd.Flags().StringVar(&cmd.portFlags.Selector, "selector", "", "Comma separated key=value selector list (e.g. release=test)")
	removePortCmd.Flags().StringVar(&cmd.portFlags.Ports, "ports", "", "Comma separated list of ports (local and remote) to remove (e.g. 8080,3000)")
	removePortCmd.Flags().BoolVar(&cmd.portFlags.RemoveAll, "all", false, "Remove all configured ports")

	removeCmd.AddCommand(removePortCmd)
}

// RunRemoveSync executes the remove sync command logic
func (cmd *RemoveCmd) RunRemoveSync(cobraCmd *cobra.Command, args []string) {
	loadConfig(&cmd.workdir, &cmd.privateConfig, &cmd.dsConfig)

	labelSelectorMap, err := parseSelectors(cmd.syncFlags.Selector)

	if err != nil {
		log.Fatalf("Error parsing selectors: %s", err.Error())
	}

	if len(labelSelectorMap) == 0 && cmd.syncFlags.RemoveAll == false && cmd.syncFlags.LocalPath == "" && cmd.syncFlags.ContainerPath == "" {
		log.Errorf("You have to specify at least one of the supported flags")
		cobraCmd.Help()

		return
	}

	newSyncPaths := make([]*v1.SyncPath, 0, len(cmd.dsConfig.SyncPaths)-1)

	for _, v := range cmd.dsConfig.SyncPaths {
		if cmd.syncFlags.RemoveAll ||
			cmd.syncFlags.LocalPath == v.LocalSubPath ||
			cmd.syncFlags.ContainerPath == v.ContainerPath ||
			isMapEqual(labelSelectorMap, v.LabelSelector) {
			continue
		}

		newSyncPaths = append(newSyncPaths, v)
	}

	cmd.dsConfig.SyncPaths = newSyncPaths

	err = config.SaveConfig(cmd.dsConfig)

	if err != nil {
		log.Fatalf("Couldn't save config file: %s", err.Error())
	}
}

// RunRemovePort executes the remove port command logic
func (cmd *RemoveCmd) RunRemovePort(cobraCmd *cobra.Command, args []string) {
	loadConfig(&cmd.workdir, &cmd.privateConfig, &cmd.dsConfig)

	labelSelectorMap, err := parseSelectors(cmd.portFlags.Selector)

	if err != nil {
		log.Fatalf("Error parsing selectors: %s", err.Error())
	}

	if len(labelSelectorMap) == 0 && cmd.portFlags.RemoveAll == false && cmd.portFlags.Ports == "" {
		log.Errorf("You have to specify at least one of the supported flags")
		cobraCmd.Help()

		return
	}

	ports := strings.Split(cmd.portFlags.Ports, ",")
	newPortForwards := make([]*v1.PortForwarding, 0, len(cmd.dsConfig.PortForwarding)-1)

OUTER:
	for _, v := range cmd.dsConfig.PortForwarding {
		if cmd.portFlags.RemoveAll ||
			isMapEqual(labelSelectorMap, v.LabelSelector) {
			continue
		}

		for _, pm := range v.PortMappings {
			if containsPort(strconv.Itoa(pm.LocalPort), ports) || containsPort(strconv.Itoa(pm.RemotePort), ports) {
				continue OUTER
			}
		}

		newPortForwards = append(newPortForwards, v)
	}

	cmd.dsConfig.PortForwarding = newPortForwards

	err = config.SaveConfig(cmd.dsConfig)

	if err != nil {
		log.Fatalf("Couldn't save config file: %s", err.Error())
	}
}

func containsPort(port string, ports []string) bool {
	for _, v := range ports {
		if strings.TrimSpace(v) == port {
			return true
		}
	}

	return false
}
