package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"k8s.io/helm/pkg/repo"

	"github.com/covexo/devspace/pkg/util/stdinutil"
	"github.com/covexo/devspace/pkg/util/tar"
	"github.com/covexo/devspace/pkg/util/yamlutil"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	helmClient "github.com/covexo/devspace/pkg/devspace/helm"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/devspace/services"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/russross/blackfriday"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
)

// AddCmd holds the information needed for the add command
type AddCmd struct {
	flags        *AddCmdFlags
	syncFlags    *addSyncCmdFlags
	portFlags    *addPortCmdFlags
	packageFlags *addPackageFlags
	dsConfig     *v1.DevSpaceConfig
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

type addPackageFlags struct {
	AppVersion   string
	ChartVersion string
	SkipQuestion bool
}

func init() {
	cmd := &AddCmd{
		flags:        &AddCmdFlags{},
		syncFlags:    &addSyncCmdFlags{},
		portFlags:    &addPortCmdFlags{},
		packageFlags: &addPackageFlags{},
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
		Short: "Add a new port forward configuration",
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

	addPackageCmd := &cobra.Command{
		Use:   "package",
		Short: "Add a helm chart",
		Long: ` 
	#######################################################
	############### devspace add package ##################
	#######################################################
	Adds an existing helm chart to the devspace
	(run 'devspace add package' to display all available 
	helm charts)
	
	Examples:
	devspace add package
	devspace add package mysql
	devspace add package mysql --app-version=5.7.14
	devspace add package mysql --chart-version=0.10.3
	#######################################################
	`,
		Run: cmd.RunAddPackage,
	}

	addPackageCmd.Flags().StringVar(&cmd.packageFlags.AppVersion, "app-version", "", "App version")
	addPackageCmd.Flags().StringVar(&cmd.packageFlags.ChartVersion, "chart-version", "", "Chart version")
	addPackageCmd.Flags().BoolVar(&cmd.packageFlags.SkipQuestion, "skip-question", false, "Skips the question to show the readme in a browser")

	addCmd.AddCommand(addPackageCmd)
}

// RunAddPackage executes the add package command logic
func (cmd *AddCmd) RunAddPackage(cobraCmd *cobra.Command, args []string) {
	kubectl, err := kubectl.NewClient()
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %v", err)
	}

	helm, err := helmClient.NewClient(kubectl, log.GetInstance(), false)
	if err != nil {
		log.Fatalf("Error initializing helm client: %v", err)
	}

	if len(args) != 1 {
		helm.PrintAllAvailableCharts()
		return
	}

	log.StartWait("Search Chart")
	repo, version, err := helm.SearchChart(args[0], cmd.packageFlags.ChartVersion, cmd.packageFlags.AppVersion)
	log.StopWait()

	if err != nil {
		log.Fatal(err)
	}

	log.Done("Chart found")

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	requirementsFile := filepath.Join(cwd, "chart", "requirements.yaml")
	_, err = os.Stat(requirementsFile)
	if os.IsNotExist(err) {
		entry := "dependencies:\n" +
			"- name: \"" + version.GetName() + "\"\n" +
			"  version: \"" + version.GetVersion() + "\"\n" +
			"  repository: \"" + repo.URL + "\"\n"

		err = ioutil.WriteFile(requirementsFile, []byte(entry), 0600)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		yamlContents := map[interface{}]interface{}{}
		err = yamlutil.ReadYamlFromFile(requirementsFile, yamlContents)
		if err != nil {
			log.Fatalf("Error parsing %s: %v", requirementsFile, err)
		}

		dependenciesArr := []interface{}{}
		if dependencies, ok := yamlContents["dependencies"]; ok {
			dependenciesArr, ok = dependencies.([]interface{})
			if ok == false {
				log.Fatalf("Error parsing %s: Key dependencies is not an array", requirementsFile)
			}
		}

		dependenciesArr = append(dependenciesArr, map[interface{}]interface{}{
			"name":       version.GetName(),
			"version":    version.GetVersion(),
			"repository": repo.URL,
		})
		yamlContents["dependencies"] = dependenciesArr

		err = yamlutil.WriteYamlToFile(yamlContents, requirementsFile)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.StartWait("Update chart dependencies")
	err = helm.UpdateDependencies(filepath.Join(cwd, "chart"))
	log.StopWait()

	if err != nil {
		log.Fatal(err)
	}

	// Check if key already exists
	valuesYaml := filepath.Join(cwd, "chart", "values.yaml")
	valuesYamlContents := map[interface{}]interface{}{}

	err = yamlutil.ReadYamlFromFile(valuesYaml, valuesYamlContents)
	if err != nil {
		log.Fatalf("Error parsing %s: %v", valuesYaml, err)
	}

	if _, ok := valuesYamlContents[version.GetName()]; ok == false {
		f, err := os.OpenFile(valuesYaml, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			log.Fatal(err)
		}

		defer f.Close()
		if _, err = f.WriteString("\n# Here you can specify the subcharts values (for more information see: https://github.com/helm/helm/blob/master/docs/chart_template_guide/subcharts_and_globals.md#overriding-values-from-a-parent-chart)\n" + version.GetName() + ": {}\n"); err != nil {
			log.Fatal(err)
		}
	}

	log.Donef("Successfully added %s as chart dependency, you can configure the package in 'chart/values.yaml'", version.GetName())
	cmd.showReadme(version)
}

func (cmd *AddCmd) showReadme(chartVersion *repo.ChartVersion) {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	if cmd.packageFlags.SkipQuestion {
		return
	}

	showReadme := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:               "Do you want to open the package README? (y|n)",
		DefaultValue:           "y",
		ValidationRegexPattern: "^(y|n)",
	})

	if showReadme == "n" {
		return
	}

	content, err := tar.ExtractSingleFileToStringTarGz(filepath.Join(cwd, "chart", "charts", chartVersion.GetName()+"-"+chartVersion.GetVersion()+".tgz"), chartVersion.GetName()+"/README.md")
	if err != nil {
		log.Fatal(err)
	}

	output := blackfriday.MarkdownCommon([]byte(content))
	f, err := os.OpenFile(filepath.Join(os.TempDir(), "Readme.html"), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err = f.Write(output)
	if err != nil {
		log.Fatal(err)
	}

	f.Close()
	open.Start(f.Name())
}

// RunAddSync executes the add sync command logic
func (cmd *AddCmd) RunAddSync(cobraCmd *cobra.Command, args []string) {
	config := configutil.GetConfig()

	if cmd.syncFlags.Selector == "" {
		cmd.syncFlags.Selector = "release=" + services.GetNameOfFirstHelmDeployment()
	}

	labelSelectorMap, err := parseSelectors(cmd.syncFlags.Selector)
	if err != nil {
		log.Fatalf("Error parsing selectors: %s", err.Error())
	}

	excludedPaths := make([]string, 0, 0)
	if cmd.syncFlags.ExcludedPaths != "" {
		excludedPathStrings := strings.Split(cmd.syncFlags.ExcludedPaths, ",")

		for _, v := range excludedPathStrings {
			excludedPath := strings.TrimSpace(v)
			excludedPaths = append(excludedPaths, excludedPath)
		}
	}

	workdir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Unable to determine current workdir: %s", err.Error())
	}

	cmd.syncFlags.LocalPath = strings.TrimPrefix(cmd.syncFlags.LocalPath, workdir)
	cmd.syncFlags.LocalPath = "./" + strings.TrimPrefix(cmd.syncFlags.LocalPath, "./")

	if cmd.syncFlags.ContainerPath[0] != '/' {
		log.Fatal("ContainerPath (--container) must start with '/'. Info: There is an issue with MINGW based terminals like git bash.")
	}

	syncConfig := append(*config.DevSpace.Sync, &v1.SyncConfig{
		ResourceType:  nil,
		LabelSelector: &labelSelectorMap,
		ContainerPath: configutil.String(cmd.syncFlags.ContainerPath),
		LocalSubPath:  configutil.String(cmd.syncFlags.LocalPath),
		ExcludePaths:  &excludedPaths,
	})

	config.DevSpace.Sync = &syncConfig

	err = configutil.SaveConfig()
	if err != nil {
		log.Fatalf("Couldn't save config file: %s", err.Error())
	}
}

// RunAddPort executes the add port command logic
func (cmd *AddCmd) RunAddPort(cobraCmd *cobra.Command, args []string) {
	if cmd.portFlags.Selector == "" {
		cmd.portFlags.Selector = "release=" + services.GetNameOfFirstHelmDeployment()
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
	config := configutil.GetConfig()

	// Check if we should add to existing port mapping
	for _, v := range *config.DevSpace.PortForwarding {
		var selectors map[string]*string

		if v.LabelSelector != nil {
			selectors = *v.LabelSelector
		} else {
			selectors = map[string]*string{}
		}

		if *v.ResourceType == cmd.portFlags.ResourceType && isMapEqual(selectors, labelSelectorMap) {
			portMap := append(*v.PortMappings, portMappings...)

			v.PortMappings = &portMap

			return
		}
	}
	portMap := append(*config.DevSpace.PortForwarding, &v1.PortForwardingConfig{
		ResourceType:  nil,
		LabelSelector: &labelSelectorMap,
		PortMappings:  &portMappings,
	})

	config.DevSpace.PortForwarding = &portMap
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
