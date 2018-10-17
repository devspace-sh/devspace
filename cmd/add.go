package cmd

import (
	"github.com/covexo/devspace/pkg/devspace/configure"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// AddCmd holds the information needed for the add command
type AddCmd struct {
	flags           *AddCmdFlags
	syncFlags       *addSyncCmdFlags
	portFlags       *addPortCmdFlags
	packageFlags    *addPackageFlags
	deploymentFlags *addDeploymentFlags
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
	Namespace     string
}

type addPortCmdFlags struct {
	ResourceType string
	Selector     string
	Namespace    string
}

type addPackageFlags struct {
	AppVersion   string
	ChartVersion string
	SkipQuestion bool
	Deployment   string
}

type addDeploymentFlags struct {
	Namespace string
	Manifests string
	Chart     string
}

func init() {
	cmd := &AddCmd{
		flags:           &AddCmdFlags{},
		syncFlags:       &addSyncCmdFlags{},
		portFlags:       &addPortCmdFlags{},
		packageFlags:    &addPackageFlags{},
		deploymentFlags: &addDeploymentFlags{},
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
	addSyncCmd.Flags().StringVar(&cmd.syncFlags.Namespace, "namespace", "", "Namespace to use")
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
	addPortCmd.Flags().StringVar(&cmd.portFlags.Namespace, "namespace", "", "Namespace to use")
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
	devspace add package mysql --chart-version=0.10.3 -d devspace-default
	#######################################################
	`,
		Run: cmd.RunAddPackage,
	}

	addPackageCmd.Flags().StringVar(&cmd.packageFlags.AppVersion, "app-version", "", "App version")
	addPackageCmd.Flags().StringVar(&cmd.packageFlags.ChartVersion, "chart-version", "", "Chart version")
	addPackageCmd.Flags().StringVarP(&cmd.packageFlags.Deployment, "deployment", "d", "", "The deployment name to use")
	addPackageCmd.Flags().BoolVar(&cmd.packageFlags.SkipQuestion, "skip-question", false, "Skips the question to show the readme in a browser")

	addCmd.AddCommand(addPackageCmd)

	addDeploymentCmd := &cobra.Command{
		Use:   "deployment",
		Short: "Add a deployment",
		Long: ` 
	#######################################################
	############# devspace add deployment #################
	#######################################################
	Add a new deployment (kubernetes manifests or 
	helm chart) to your devspace, that will be deployed
	
	Examples:
	devspace add deployment my-deployment --chart=chart/
	devspace add deployment my-deployment --manifests=kube/pod.yaml
	devspace add deployment my-deployment --manifests=kube/* --namespace=devspace
	#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunAddDeployment,
	}

	addDeploymentCmd.Flags().StringVar(&cmd.deploymentFlags.Namespace, "namespace", "", "The namespace to use for deploying")
	addDeploymentCmd.Flags().StringVar(&cmd.deploymentFlags.Manifests, "manifests", "", "The kubernetes manifests to deploy (glob pattern are allowed, comma separated)")
	addDeploymentCmd.Flags().StringVar(&cmd.deploymentFlags.Chart, "chart", "", "The helm chart to deploy")

	addCmd.AddCommand(addDeploymentCmd)
}

// RunAddPackage executes the add package command logic
func (cmd *AddCmd) RunAddPackage(cobraCmd *cobra.Command, args []string) {
	name, chartPath, err := configure.AddPackage(cmd.packageFlags.SkipQuestion, cmd.packageFlags.AppVersion, cmd.packageFlags.ChartVersion, cmd.packageFlags.Deployment, args, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	log.Donef("Successfully added %s as chart dependency, you can configure the package in '%s/values.yaml'", name, chartPath)
}

// RunAddDeployment executes the add deployment command logic
func (cmd *AddCmd) RunAddDeployment(cobraCmd *cobra.Command, args []string) {
	err := configure.AddDeployment(args[0], cmd.deploymentFlags.Namespace, cmd.deploymentFlags.Manifests, cmd.deploymentFlags.Chart)
	if err != nil {
		log.Fatal(err)
	}

	log.Donef("Successfully added %s as new deployment", args[0])
}

// RunAddSync executes the add sync command logic
func (cmd *AddCmd) RunAddSync(cobraCmd *cobra.Command, args []string) {
	err := configure.AddSyncPath(cmd.syncFlags.LocalPath, cmd.syncFlags.ContainerPath, cmd.syncFlags.Namespace, cmd.syncFlags.Selector, cmd.syncFlags.ExcludedPaths)
	if err != nil {
		log.Fatalf("Error adding sync path: %v", err)
	}
}

// RunAddPort executes the add port command logic
func (cmd *AddCmd) RunAddPort(cobraCmd *cobra.Command, args []string) {
	err := configure.AddPort(cmd.portFlags.Namespace, cmd.portFlags.Selector, args)
	if err != nil {
		log.Fatal(err)
	}
}
