package list

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy"
	deployComponent "github.com/devspace-cloud/devspace/pkg/devspace/deploy/component"
	deployHelm "github.com/devspace-cloud/devspace/pkg/devspace/deploy/helm"
	deployKubectl "github.com/devspace-cloud/devspace/pkg/devspace/deploy/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type deploymentsCmd struct{}

func newDeploymentsCmd() *cobra.Command {
	cmd := &deploymentsCmd{}

	return &cobra.Command{
		Use:   "deployments",
		Short: "Lists and shows the status of all deployments",
		Long: `
#######################################################
############# devspace list deployments ###############
#######################################################
Shows the status of all deployments
#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunDeploymentsStatus,
	}
}

// RunDeploymentsStatus executes the devspace status deployments command logic
func (cmd *deploymentsCmd) RunDeploymentsStatus(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find any devspace configuration. Please run `devspace init`")
	}

	var values [][]string
	var headerValues = []string{
		"NAME",
		"TYPE",
		"DEPLOY",
		"STATUS",
	}

	config := configutil.GetConfig()
	kubectl, err := kubectl.NewClient()
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %s", err.Error())
	}

	if config.Deployments != nil {
		for _, deployConfig := range *config.Deployments {
			var deployClient deploy.Interface

			// Delete kubectl engine
			if deployConfig.Kubectl != nil {
				deployClient, err = deployKubectl.New(kubectl, deployConfig, log.GetInstance())
				if err != nil {
					log.Warnf("Unable to create kubectl deploy config for %s: %v", *deployConfig.Name, err)
					continue
				}
			} else if deployConfig.Helm != nil {
				deployClient, err = deployHelm.New(kubectl, deployConfig, log.GetInstance())
				if err != nil {
					log.Warnf("Unable to create helm deploy config for %s: %v", *deployConfig.Name, err)
					continue
				}
			} else if deployConfig.Component != nil {
				deployClient, err = deployComponent.New(kubectl, deployConfig, log.GetInstance())
				if err != nil {
					log.Warnf("Unable to create component deploy config for %s: %v", *deployConfig.Name, err)
					continue
				}
			}

			status, err := deployClient.Status()
			if err != nil {
				log.Warnf("Error retrieving status for deployment %s: %v", *deployConfig.Name, err)
			}

			values = append(values, []string{
				status.Name,
				status.Type,
				status.Target,
				status.Status,
			})
		}
	}

	log.PrintTable(headerValues, values)
}
