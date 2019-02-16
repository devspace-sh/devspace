package status

import (
	"github.com/covexo/devspace/pkg/devspace/cloud"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/deploy"
	deployHelm "github.com/covexo/devspace/pkg/devspace/deploy/helm"
	deployKubectl "github.com/covexo/devspace/pkg/devspace/deploy/kubectl"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type deploymentsCmd struct{}

func newDeploymentsCmd() *cobra.Command {
	cmd := &deploymentsCmd{}

	return &cobra.Command{
		Use:   "deployments",
		Short: "Shows the deployments status",
		Long: `
	#######################################################
	############ devspace status deployments ##############
	#######################################################
	Shows the devspace status
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
		"TYPE",
		"STATUS",
		"NAMESPACE",
		"INFO",
	}
	config := configutil.GetConfig()

	// Configure cloud provider
	err = cloud.Configure(log.GetInstance())
	if err != nil {
		log.Fatalf("Unable to configure cloud provider: %v", err)
	}

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
			} else {
				deployClient, err = deployHelm.New(kubectl, deployConfig, log.GetInstance())
				if err != nil {
					log.Warnf("Unable to create helm deploy config for %s: %v", *deployConfig.Name, err)
					continue
				}
			}

			addValues, err := deployClient.Status()
			if err != nil {
				log.Warnf("Error retrieving status for deployment %s: %v", *deployConfig.Name, err)
			}

			values = append(values, addValues...)
		}
	}

	log.PrintTable(headerValues, values)
}
