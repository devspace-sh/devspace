package update

import (
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/generator"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/spf13/cobra"
)

// chartCmd holds the cmd flags
type chartCmd struct {
	Deployment string
	Force      bool
}

// newChartCmd creates a new command
func newChartCmd() *cobra.Command {
	cmd := &chartCmd{}

	chartCmd := &cobra.Command{
		Use:   "chart",
		Short: "Updates the chart if necessary to the current version",
		Long: `
#######################################################
############### devspace update chart ################
#######################################################
Updates the devspace chart to the newest version

Examples:
devspace update chart
devspace update chart --force
devspace update chart --deployment=my-deployment
#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunChart,
	}

	chartCmd.Flags().StringVar(&cmd.Deployment, "deployment", "", "The deployment name to use")
	chartCmd.Flags().BoolVar(&cmd.Force, "force", false, "Force chart update")

	return chartCmd
}

// RunChart executes the functionality "devspace update chart"
func (cmd *chartCmd) RunChart(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	// Get config
	config := configutil.GetConfig()
	if config.Deployments == nil || len(*config.Deployments) == 0 {
		log.Fatal("No deployment specified")
	}

	// get all helm deployments
	var helmDeployments []*latest.DeploymentConfig
	for _, deploy := range *config.Deployments {
		if deploy.Helm != nil {
			helmDeployments = append(helmDeployments, deploy)
		}
	}
	if len(helmDeployments) == 0 {
		log.Fatal("There is no helm deployment specified in configuration")
	}

	helmDeployment := helmDeployments[0]
	if cmd.Deployment != "" {
		found := false
		for _, deploy := range helmDeployments {
			if *deploy.Name == cmd.Deployment {
				helmDeployment = deploy
				found = true
				break
			}
		}

		if found == false {
			log.Fatalf("Couldn't find a deployment with name %s", cmd.Deployment)
		}
	} else if len(helmDeployments) > 1 {
		deployments := []string{}
		for _, deploy := range helmDeployments {
			deployments = append(deployments, *deploy.Name)
		}

		deploymentName := survey.Question(&survey.QuestionOptions{
			Question: "Select a deployment",
			Options:  deployments,
		})

		for _, deploy := range helmDeployments {
			if *deploy.Name == deploymentName {
				helmDeployment = deploy
				break
			}
		}
	}

	// Check if the chart is there
	chartPath := *helmDeployment.Helm.Chart.Name
	_, err = os.Stat(chartPath)
	if err != nil {
		log.Fatalf("Chart %s is not a local chart path", chartPath)
	}

	// Create chart generator
	chartGenerator, err := generator.NewChartGenerator(chartPath)
	if err != nil {
		log.Fatalf("Error initializing chart generator: %v", err)
	}

	// Update the chart
	err = chartGenerator.Update(cmd.Force)
	if err != nil {
		log.Fatalf("Error updating chart %s: %v", chartPath, err)
	}

	log.Donef("Successfully updated chart %s", chartPath)
}
