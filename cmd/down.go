package cmd

import (
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	helmClient "github.com/covexo/devspace/pkg/devspace/deploy/helm"
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
	var err error
	var client *helmClient.ClientWrapper

	config := configutil.GetConfig()

	if config.DevSpace.Deploy != nil {
		for _, deployConfig := range *config.DevSpace.Deploy {
			// Delete kubectl engine
			if deployConfig.Engine != nil && deployConfig.Engine.Kubectl != nil {
				kubectlDeployConfig, err := deployKubectl.New(config, deployConfig)
				if err != nil {
					log.Warnf("Unable to create kubectl deploy config: %v", err)
					continue
				}

				log.StartWait("Deleting kubectl deployment")
				err = kubectlDeployConfig.Delete()
				log.StopWait()
				if err != nil {
					log.Warnf("Error deleting kubectl manifests: %v", err)
				}

				log.Donef("Successfully deleted kubectl deployment")
			} else {
				// Delete with helm engine
				defaultReleaseName := configutil.GetDefaultDevSpaceDefaultReleaseName(config)

				releaseName := defaultReleaseName
				if deployConfig.Engine != nil && deployConfig.Engine.Helm != nil && deployConfig.Engine.Helm.Release != nil {
					releaseName = deployConfig.Engine.Helm.Release
				}

				if client == nil {
					isDeployed := helmClient.IsTillerDeployed(kubectl)
					if isDeployed == false {
						continue
					}

					client, err = helmClient.NewClient(kubectl, false)
					if err != nil {
						log.Warnf("Unable to initialize helm client: %s", err.Error())
						continue
					}
				}

				log.StartWait("Deleting release " + *releaseName)
				res, err := client.DeleteRelease(*releaseName, true)
				log.StopWait()

				if res != nil && res.Info != "" {
					log.Donef("Successfully deleted release %s: %s", releaseName, res.Info)
				} else if err != nil {
					log.Warnf("Error deleting release %s: %s", releaseName, err.Error())
				} else {
					log.Donef("Successfully deleted release %s", releaseName)
				}
			}
		}
	}
}
