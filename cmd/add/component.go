package add

import (
	"github.com/spf13/cobra"
)

type componentCmd struct {
	Deployment string
}

func newComponentCmd() *cobra.Command {
	cmd := &componentCmd{}

	componentCmd := &cobra.Command{
		Use:   "component [name]",
		Short: "Add a component to the chart",
		Long: ` 
#######################################################
############## devspace add component #################
#######################################################
Adds a component to the chart. 
Run 'devspace list available-components' to see all available components

Examples:
devspace add component mysql
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunAddComponent,
	}

	componentCmd.Flags().StringVarP(&cmd.Deployment, "deployment", "d", "", "The deployment name to use")

	return componentCmd
}

// RunAddPackage executes the add package command logic
func (cmd *componentCmd) RunAddComponent(cobraCmd *cobra.Command, args []string) {
	// Set config root
	/*configExists, err := configutil.SetDevSpaceRoot()
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

		deploymentName := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
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

	// Add the component to the chart
	err = chart.AddComponent(*helmDeployment.Helm.ChartPath, args[0])
	if err != nil {
		log.Fatalf("Error adding component %s: %v", args[0], err)
	}

	log.Donef("Successfully added the component %s\nRun:\n- `%s` to update the application", args[0], ansi.Color("devspace deploy", "white+b"))*/
}
