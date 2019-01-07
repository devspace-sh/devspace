package cmd

import (
	"os"
	"path/filepath"
	"strconv"

	cloudCmd "github.com/covexo/devspace/cmd/cloud"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/yamlutil"
	"github.com/spf13/cobra"
)

// ListCmd holds the information needed for the list command
type ListCmd struct {
	flags *ListCmdFlags
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

	listPackageCmd := &cobra.Command{
		Use:   "package",
		Short: "Lists all added packages",
		Long: `
	#######################################################
	############### devspace list package #################
	#######################################################
	Lists the packages that were added to the DevSpace
	#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunListPackage,
	}

	listCmd.AddCommand(listPackageCmd)

	listServiceCmd := &cobra.Command{
		Use:   "service",
		Short: "Lists all services",
		Long: `
	#######################################################
	############### devspace list service #################
	#######################################################
	Lists the service that are defined in the DevSpace
	#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunListService,
	}

	listCmd.AddCommand(listServiceCmd)

	// Add cloud commands
	listCmd.AddCommand(cloudCmd.Cmd)
}

// RunListPackage runs the list sync command logic
func (cmd *ListCmd) RunListPackage(cobraCmd *cobra.Command, args []string) {
	headerColumnNames := []string{
		"Name",
		"Version",
		"Repository",
	}
	values := [][]string{}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	requirementsFile := filepath.Join(cwd, "chart", "requirements.yaml")
	_, err = os.Stat(requirementsFile)
	if os.IsNotExist(err) == false {
		yamlContents := map[interface{}]interface{}{}
		err = yamlutil.ReadYamlFromFile(requirementsFile, yamlContents)
		if err != nil {
			log.Fatalf("Error parsing %s: %v", requirementsFile, err)
		}

		if dependencies, ok := yamlContents["dependencies"]; ok {
			if dependenciesArr, ok := dependencies.([]interface{}); ok {
				for _, dependency := range dependenciesArr {
					if dependencyMap, ok := dependency.(map[interface{}]interface{}); ok {
						values = append(values, []string{
							dependencyMap["name"].(string),
							dependencyMap["version"].(string),
							dependencyMap["repository"].(string),
						})
					}
				}
			}
		}
	}

	log.PrintTable(headerColumnNames, values)
}

// RunListService runs the list service command logic
func (cmd *ListCmd) RunListService(cobraCmd *cobra.Command, args []string) {
	config := configutil.GetConfig()

	if config.DevSpace.Services == nil || len(*config.DevSpace.Services) == 0 {
		log.Info("No services are configured. Run `devspace add service` to add new service\n")
		return
	}

	headerColumnNames := []string{
		"Name",
		"Namespace",
		"Type",
		"Selector",
		"Container",
	}

	services := make([][]string, 0, len(*config.DevSpace.Services))

	// Transform values into string arrays
	for _, value := range *config.DevSpace.Services {
		selector := ""
		for k, v := range *value.LabelSelector {
			if len(selector) > 0 {
				selector += ", "
			}

			selector += k + "=" + *v
		}

		resourceType := "pod"
		if value.ResourceType != nil {
			resourceType = *value.ResourceType
		}

		// TODO: should we skip this error?
		namespace, _ := configutil.GetDefaultNamespace(config)
		if value.Namespace != nil {
			namespace = *value.Namespace
		}

		containerName := ""
		if value.ContainerName != nil {
			containerName = *value.ContainerName
		}

		services = append(services, []string{
			*value.Name,
			namespace,
			resourceType,
			selector,
			containerName,
		})
	}

	log.PrintTable(headerColumnNames, services)
}

// RunListSync runs the list sync command logic
func (cmd *ListCmd) RunListSync(cobraCmd *cobra.Command, args []string) {
	config := configutil.GetConfig()

	if config.DevSpace.Sync == nil || len(*config.DevSpace.Sync) == 0 {
		log.Info("No sync paths are configured. Run `devspace add sync` to add new sync path\n")
		return
	}

	headerColumnNames := []string{
		"Service",
		"Selector",
		"Local Path",
		"Container Path",
		"Excluded Paths",
	}

	syncPaths := make([][]string, 0, len(*config.DevSpace.Sync))

	// Transform values into string arrays
	for _, value := range *config.DevSpace.Sync {
		service := ""
		selector := ""

		if value.Service != nil {
			service = *value.Service
		} else {
			for k, v := range *value.LabelSelector {
				if len(selector) > 0 {
					selector += ", "
				}

				selector += k + "=" + *v
			}
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
			service,
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
	config := configutil.GetConfig()

	if config.DevSpace.Ports == nil || len(*config.DevSpace.Ports) == 0 {
		log.Info("No ports are forwarded. Run `devspace add port` to add a port that should be forwarded\n")
		return
	}

	headerColumnNames := []string{
		"Service",
		"Type",
		"Selector",
		"Ports (Local:Remote)",
	}

	portForwards := make([][]string, 0, len(*config.DevSpace.Ports))

	// Transform values into string arrays
	for _, value := range *config.DevSpace.Ports {
		service := ""
		selector := ""

		if value.Service != nil {
			service = *value.Service
		} else {
			for k, v := range *value.LabelSelector {
				if len(selector) > 0 {
					selector += ", "
				}

				selector += k + "=" + *v
			}
		}

		portMappings := ""
		for _, v := range *value.PortMappings {
			if len(portMappings) > 0 {
				portMappings += ", "
			}

			portMappings += strconv.Itoa(*v.LocalPort) + ":" + strconv.Itoa(*v.RemotePort)
		}

		resourceType := "pod"
		if value.ResourceType != nil {
			resourceType = *value.ResourceType
		}

		portForwards = append(portForwards, []string{
			service,
			resourceType,
			selector,
			portMappings,
		})
	}

	log.PrintTable(headerColumnNames, portForwards)
}
