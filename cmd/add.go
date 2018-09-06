package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// AddCmd holds the information needed for the add command
type AddCmd struct {
	flags         *AddCmdFlags
	syncFlags     *addSyncCmdFlags
	portFlags     *addPortCmdFlags
	dsConfig      *v1.DevSpaceConfig
	privateConfig *v1.PrivateConfig
	workdir       string
}

// AddCmdFlags holds the possible flags for the add command
type AddCmdFlags struct {
}

type addSyncCmdFlags struct {
	ResourceType  string
	Selector      string
	LocalPath     string
	ContainerPath string
	ExcludedPaths string
}

type addPortCmdFlags struct {
	ResourceType string
	Selector     string
}

func init() {
	cmd := &AddCmd{
		flags:     &AddCmdFlags{},
		syncFlags: &addSyncCmdFlags{},
		portFlags: &addPortCmdFlags{},
	}

	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Change the devspace configuration",
		Long: `
	#######################################################
	#################### devspace add #####################
	#######################################################
	You can change the following configuration with the
	add command:
	
	* Sync paths (sync)
	* Forwarded ports (port)
	#######################################################
	`,
		Args: cobra.NoArgs,
	}

	rootCmd.AddCommand(addCmd)

	addSyncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Add a sync path to the devspace",
		Long: `
	#######################################################
	################# devspace add sync ###################
	#######################################################
	Add a sync path to the devspace

	How to use:
	devspace add sync --local=app --container=/app
	#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunAddSync,
	}

	addCmd.AddCommand(addSyncCmd)

	addSyncCmd.Flags().StringVar(&cmd.syncFlags.ResourceType, "resource-type", "pod", "Selected resource type")
	addSyncCmd.Flags().StringVar(&cmd.syncFlags.Selector, "selector", "", "Comma separated key=value selector list (e.g. release=test)")
	addSyncCmd.Flags().StringVar(&cmd.syncFlags.LocalPath, "local", "", "Relative local path")
	addSyncCmd.Flags().StringVar(&cmd.syncFlags.ContainerPath, "container", "", "Absolute container path")
	addSyncCmd.Flags().StringVar(&cmd.syncFlags.ExcludedPaths, "exclude", "", "Comma separated list of paths to exclude (e.g. node_modules/,bin,*.exe)")

	addSyncCmd.MarkFlagRequired("local")
	addSyncCmd.MarkFlagRequired("container")

	addPortCmd := &cobra.Command{
		Use:   "port",
		Short: "Lists port forwarding configuration",
		Long: `
	#######################################################
	################ devspace add port ####################
	#######################################################
	Add a new port mapping that should be forwarded to
	the devspace (format is local:remote comma separated):
	devspace add port 8080:80,3000
	#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunAddPort,
	}

	addPortCmd.Flags().StringVar(&cmd.portFlags.ResourceType, "resource-type", "pod", "Selected resource type")
	addPortCmd.Flags().StringVar(&cmd.portFlags.Selector, "selector", "", "Comma separated key=value selector list (e.g. release=test)")

	addCmd.AddCommand(addPortCmd)
}

// RunAddSync executes the add sync command logic
func (cmd *AddCmd) RunAddSync(cobraCmd *cobra.Command, args []string) {
	config := configutil.GetConfig(false)

	if cmd.syncFlags.Selector == "" {
		cmd.syncFlags.Selector = "release=" + *config.DevSpace.Release.Name
	}

	labelSelectorMap, err := parseSelectors(cmd.syncFlags.Selector)

	if err != nil {
		log.Fatalf("Error parsing selectors: %s", err.Error())
	}

	excludedPaths := make([]*string, 0, 0)

	if cmd.syncFlags.ExcludedPaths != "" {
		excludedPathStrings := strings.Split(cmd.syncFlags.ExcludedPaths, ",")

		for _, v := range excludedPathStrings {
			excludedPath := strings.TrimSpace(v)
			excludedPaths = append(excludedPaths, &excludedPath)
		}
	}

	config.DevSpace.Sync = append(config.DevSpace.Sync, &v1.SyncConfig{
		ResourceType:  configutil.String(cmd.syncFlags.ResourceType),
		LabelSelector: labelSelectorMap,
		ContainerPath: configutil.String(cmd.syncFlags.ContainerPath),
		LocalSubPath:  configutil.String(cmd.syncFlags.LocalPath),
		ExcludeRegex:  excludedPaths,
	})

	err = configutil.SaveConfig()

	if err != nil {
		log.Fatalf("Couldn't save config file: %s", err.Error())
	}
}

// RunAddPort executes the add port command logic
func (cmd *AddCmd) RunAddPort(cobraCmd *cobra.Command, args []string) {
	config := configutil.GetConfig(false)

	if cmd.portFlags.Selector == "" {
		cmd.portFlags.Selector = "release=" + *config.DevSpace.Release.Name
	}

	labelSelectorMap, err := parseSelectors(cmd.portFlags.Selector)

	if err != nil {
		log.Fatalf("Error parsing selectors: %s", err.Error())
	}

	portMappings, err := parsePortMappings(args[0])

	if err != nil {
		log.Fatalf("Error parsing port mappings: %s", err.Error())
	}

	cmd.insertOrReplacePortMapping(labelSelectorMap, portMappings)

	err = configutil.SaveConfig()

	if err != nil {
		log.Fatalf("Couldn't save config file: %s", err.Error())
	}
}

func (cmd *AddCmd) insertOrReplacePortMapping(labelSelectorMap map[string]*string, portMappings []*v1.PortMapping) {
	config := configutil.GetConfig(false)

	// Check if we should add to existing port mapping
	for _, v := range config.DevSpace.PortForwarding {
		if *v.ResourceType == cmd.portFlags.ResourceType && isMapEqual(v.LabelSelector, labelSelectorMap) {
			v.PortMappings = append(v.PortMappings, portMappings...)

			return
		}
	}

	config.DevSpace.PortForwarding = append(config.DevSpace.PortForwarding, &v1.PortForwardingConfig{
		ResourceType:  configutil.String(cmd.portFlags.ResourceType),
		LabelSelector: labelSelectorMap,
		PortMappings:  portMappings,
	})
}

func isMapEqual(map1 map[string]*string, map2 map[string]*string) bool {
	if len(map1) != len(map2) {
		return false
	}

	for k, v := range map1 {
		if *map2[k] != *v {
			return false
		}
	}

	return true
}

func parsePortMappings(portMappingsString string) ([]*v1.PortMapping, error) {
	portMappings := make([]*v1.PortMapping, 0, 1)
	portMappingsSplitted := strings.Split(portMappingsString, ",")

	for _, v := range portMappingsSplitted {
		portMapping := strings.Split(v, ":")

		if len(portMapping) != 1 && len(portMapping) != 2 {
			return nil, fmt.Errorf("Error parsing port mapping: %s", v)
		}

		portMappingStruct := &v1.PortMapping{}
		firstPort, err := strconv.Atoi(portMapping[0])

		if err != nil {
			return nil, err
		}

		if len(portMapping) == 1 {
			portMappingStruct.LocalPort = &firstPort

			portMappingStruct.RemotePort = portMappingStruct.LocalPort
		} else {
			portMappingStruct.LocalPort = &firstPort

			secondPort, err := strconv.Atoi(portMapping[1])

			if err != nil {
				return nil, err
			}
			portMappingStruct.RemotePort = &secondPort
		}

		portMappings = append(portMappings, portMappingStruct)
	}

	return portMappings, nil
}

func parseSelectors(selectorString string) (map[string]*string, error) {
	selectorMap := make(map[string]*string)

	if selectorString == "" {
		return selectorMap, nil
	}

	selectors := strings.Split(selectorString, ",")

	for _, v := range selectors {
		keyValue := strings.Split(v, "=")

		if len(keyValue) != 2 {
			return nil, fmt.Errorf("Wrong selector format: %s", selectorString)
		}
		selector := strings.TrimSpace(keyValue[1])
		selectorMap[strings.TrimSpace(keyValue[0])] = &selector
	}

	return selectorMap, nil
}
