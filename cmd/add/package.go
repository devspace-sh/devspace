package add

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/configure"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type packageCmd struct {
	AppVersion   string
	ChartVersion string
	SkipQuestion bool
	Deployment   string
}

func newPackageCmd() *cobra.Command {
	cmd := &packageCmd{}

	addPackageCmd := &cobra.Command{
		Use:   "package",
		Short: "Add a helm chart",
		Long: ` 
#######################################################
############### devspace add package ##################
#######################################################
Adds a Helm chart as dependeny to the project chart
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

	addPackageCmd.Flags().StringVar(&cmd.AppVersion, "app-version", "", "App version")
	addPackageCmd.Flags().StringVar(&cmd.ChartVersion, "chart-version", "", "Chart version")
	addPackageCmd.Flags().StringVarP(&cmd.Deployment, "deployment", "d", "", "The deployment name to use")
	addPackageCmd.Flags().BoolVar(&cmd.SkipQuestion, "skip-question", false, "Skips the question to show the readme in a browser")

	return addPackageCmd
}

// RunAddPackage executes the add package command logic
func (cmd *packageCmd) RunAddPackage(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	err = configure.AddPackage(cmd.SkipQuestion, cmd.AppVersion, cmd.ChartVersion, cmd.Deployment, args, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	log.Donef("Successfully added the package")
}
