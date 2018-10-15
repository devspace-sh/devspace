package cmd

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	helmClient "github.com/covexo/devspace/pkg/devspace/helm"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/yamlutil"
	"github.com/spf13/cobra"
)

// RemoveCmd holds the information needed for the remove command
type RemoveCmd struct {
	syncFlags    *removeSyncCmdFlags
	portFlags    *removePortCmdFlags
	packageFlags *removePackageCmdFlags
	workdir      string
}

type removeSyncCmdFlags struct {
	Selector      string
	LocalPath     string
	ContainerPath string
	RemoveAll     bool
}

type removePortCmdFlags struct {
	Selector  string
	RemoveAll bool
}

type removePackageCmdFlags struct {
	RemoveAll bool
}

func init() {
	cmd := &RemoveCmd{
		syncFlags:    &removeSyncCmdFlags{},
		portFlags:    &removePortCmdFlags{},
		packageFlags: &removePackageCmdFlags{},
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
		Args: cobra.NoArgs,
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
		Args: cobra.NoArgs,
		Run:  cmd.RunRemoveSync,
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
	devspace remove port 8080,3000
	devspace remove port --selector=release=test
	devspace remove port --all
	#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		Run:  cmd.RunRemovePort,
	}

	removePortCmd.Flags().StringVar(&cmd.portFlags.Selector, "selector", "", "Comma separated key=value selector list (e.g. release=test)")
	removePortCmd.Flags().BoolVar(&cmd.portFlags.RemoveAll, "all", false, "Remove all configured ports")

	removeCmd.AddCommand(removePortCmd)

	removePackageCmd := &cobra.Command{
		Use:   "package",
		Short: "Removes forwarded ports from a devspace",
		Long: `
	#######################################################
	############## devspace remove package ################
	#######################################################
	Removes a package from the devspace:
	devspace remove package mysql
	#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		Run:  cmd.RunRemovePackage,
	}

	removePackageCmd.Flags().BoolVar(&cmd.packageFlags.RemoveAll, "all", false, "Remove all packages")
	removeCmd.AddCommand(removePackageCmd)
}

// RunRemovePackage executes the remove package command logic
func (cmd *RemoveCmd) RunRemovePackage(cobraCmd *cobra.Command, args []string) {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	if len(args) == 0 && cmd.packageFlags.RemoveAll == false {
		log.Fatal("You need to specify a package name or the --all flag")
	}

	requirementsPath := filepath.Join(cwd, "chart", "requirements.yaml")
	yamlContents := map[interface{}]interface{}{}

	err = yamlutil.ReadYamlFromFile(requirementsPath, yamlContents)
	if err != nil {
		log.Fatal(err)
	}

	if dependencies, ok := yamlContents["dependencies"]; ok {
		dependenciesArr, ok := dependencies.([]interface{})
		if ok == false {
			log.Fatalf("Error parsing yaml: %v", dependencies)
		}

		if cmd.packageFlags.RemoveAll == false {
			for key, dependency := range dependenciesArr {
				dependencyMap, ok := dependency.(map[interface{}]interface{})
				if ok == false {
					log.Fatalf("Error parsing yaml: %v", dependencies)
				}

				if name, ok := dependencyMap["name"]; ok {
					if name == args[0] {
						dependenciesArr = append(dependenciesArr[:key], dependenciesArr[key+1:]...)
						yamlContents["dependencies"] = dependenciesArr

						cmd.rebuildDependencies(yamlContents)
						break
					}
				}
			}

			log.Donef("Successfully removed dependency %s", args[0])
			return
		}

		yamlContents["dependencies"] = []interface{}{}

		cmd.rebuildDependencies(yamlContents)
		log.Done("Successfully removed all dependencies")
		return
	}

	log.Done("No dependencies found")
}

func (cmd *RemoveCmd) rebuildDependencies(newYamlContents map[interface{}]interface{}) {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	err = yamlutil.WriteYamlToFile(newYamlContents, filepath.Join(cwd, "chart", "requirements.yaml"))
	if err != nil {
		log.Fatal(err)
	}

	// Rebuild dependencies
	kubectl, err := kubectl.NewClient()
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %v", err)
	}

	helm, err := helmClient.NewClient(kubectl, false)
	if err != nil {
		log.Fatalf("Error initializing helm client: %v", err)
	}

	log.StartWait("Update chart dependencies")
	err = helm.UpdateDependencies(filepath.Join(cwd, "chart"))
	log.StopWait()

	if err != nil {
		log.Fatal(err)
	}
}

// RunRemoveSync executes the remove sync command logic
func (cmd *RemoveCmd) RunRemoveSync(cobraCmd *cobra.Command, args []string) {
	config := configutil.GetConfig()
	labelSelectorMap, err := parseSelectors(cmd.syncFlags.Selector)

	if err != nil {
		log.Fatalf("Error parsing selectors: %s", err.Error())
	}

	if len(labelSelectorMap) == 0 && cmd.syncFlags.RemoveAll == false && cmd.syncFlags.LocalPath == "" && cmd.syncFlags.ContainerPath == "" {
		log.Errorf("You have to specify at least one of the supported flags")
		cobraCmd.Help()

		return
	}

	newSyncPaths := make([]*v1.SyncConfig, 0, len(*config.DevSpace.Sync)-1)

	for _, v := range *config.DevSpace.Sync {
		if cmd.syncFlags.RemoveAll ||
			cmd.syncFlags.LocalPath == *v.LocalSubPath ||
			cmd.syncFlags.ContainerPath == *v.ContainerPath ||
			isMapEqual(labelSelectorMap, *v.LabelSelector) {
			continue
		}

		newSyncPaths = append(newSyncPaths, v)
	}

	config.DevSpace.Sync = &newSyncPaths

	err = configutil.SaveConfig()

	if err != nil {
		log.Fatalf("Couldn't save config file: %s", err.Error())
	}
}

// RunRemovePort executes the remove port command logic
func (cmd *RemoveCmd) RunRemovePort(cobraCmd *cobra.Command, args []string) {
	config := configutil.GetConfig()

	labelSelectorMap, err := parseSelectors(cmd.portFlags.Selector)

	if err != nil {
		log.Fatalf("Error parsing selectors: %s", err.Error())
	}

	argPorts := ""

	if len(args) == 1 {
		argPorts = args[0]
	}

	if len(labelSelectorMap) == 0 && cmd.portFlags.RemoveAll == false && argPorts == "" {
		log.Errorf("You have to specify at least one of the supported flags")
		cobraCmd.Help()

		return
	}

	ports := strings.Split(argPorts, ",")
	newPortForwards := make([]*v1.PortForwardingConfig, 0, len(*config.DevSpace.PortForwarding)-1)

OUTER:
	for _, v := range *config.DevSpace.PortForwarding {
		if cmd.portFlags.RemoveAll ||
			isMapEqual(labelSelectorMap, *v.LabelSelector) {
			continue
		}

		for _, pm := range *v.PortMappings {
			if containsPort(strconv.Itoa(*pm.LocalPort), ports) || containsPort(strconv.Itoa(*pm.RemotePort), ports) {
				continue OUTER
			}
		}

		newPortForwards = append(newPortForwards, v)
	}

	config.DevSpace.PortForwarding = &newPortForwards

	err = configutil.SaveConfig()

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
