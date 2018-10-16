package cmd

import (
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/deploy"
	deployHelm "github.com/covexo/devspace/pkg/devspace/deploy/helm"
	deployKubectl "github.com/covexo/devspace/pkg/devspace/deploy/kubectl"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/log"
	"k8s.io/client-go/kubernetes"

	"github.com/spf13/cobra"
)

// DownCmd holds the required data for the down cmd
type DownCmd struct {
	flags *DownCmdFlags
}

// DownCmdFlags holds the possible down cmd flags
type DownCmdFlags struct {
}

func init() {
	cmd := &DownCmd{
		flags: &DownCmdFlags{},
	}

	cobraCmd := &cobra.Command{
		Use:   "down",
		Short: "Shutdown your DevSpace",
		Long: `
#######################################################
################### devspace down #####################
#######################################################
Stops your DevSpace by removing the release via helm.
If you want to remove all DevSpace related data from
your project, use: devspace reset
#######################################################`,
		Args: cobra.NoArgs,
		Run:  cmd.Run,
	}
	rootCmd.AddCommand(cobraCmd)
}

// Run executes the down command logic
func (cmd *DownCmd) Run(cobraCmd *cobra.Command, args []string) {
	log.StartFileLogging()
	kubectl, err := kubectl.NewClient()
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %s", err.Error())
	}

	deleteDevSpace(kubectl)
}

func deleteDevSpace(kubectl *kubernetes.Clientset) {
	config := configutil.GetConfig()

	if config.DevSpace.Deployments != nil {
		for _, deployConfig := range *config.DevSpace.Deployments {
			var err error
			var deployClient deploy.Interface

			// Delete kubectl engine
			if deployConfig.Kubectl != nil {
				deployClient, err = deployKubectl.New(kubectl, deployConfig, log.GetInstance())
				if err != nil {
					log.Warnf("Unable to create kubectl deploy config: %v", err)
					continue
				}
			} else {
				deployClient, err = deployHelm.New(kubectl, deployConfig, log.GetInstance())
				if err != nil {
					log.Warnf("Unable to create helm deploy config: %v", err)
					continue
				}
			}

			log.StartWait("Deleting deployment %s" + *deployConfig.Name)
			err = deployClient.Delete()
			log.StopWait()
			if err != nil {
				log.Warnf("Error deleting deployment %s: %v", *deployConfig.Name, err)
			}

			log.Donef("Successfully deleted deployment %s", *deployConfig.Name)
		}
	}
}
