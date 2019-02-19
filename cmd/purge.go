package cmd

import (
	"strings"

	"github.com/covexo/devspace/pkg/devspace/cloud"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/deploy"
	deployHelm "github.com/covexo/devspace/pkg/devspace/deploy/helm"
	deployKubectl "github.com/covexo/devspace/pkg/devspace/deploy/kubectl"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/log"
	"k8s.io/client-go/kubernetes"

	"github.com/spf13/cobra"
)

// PurgeCmd holds the required data for the purge cmd
type PurgeCmd struct {
	flags *PurgeCmdFlags
}

// PurgeCmdFlags holds the possible down cmd flags
type PurgeCmdFlags struct {
	config     string
	deployment string
}

func init() {
	cmd := &PurgeCmd{
		flags: &PurgeCmdFlags{},
	}

	cobraCmd := &cobra.Command{
		Use:   "purge",
		Short: "Delete the devspace",
		Long: `
#######################################################
################### devspace purge ####################
#######################################################
Deletes the deployed devspace. 
Warning: will delete everything that is defined in the 
chart, including persistent volume claims!
#######################################################`,
		Args: cobra.NoArgs,
		Run:  cmd.Run,
	}

	cobraCmd.Flags().StringVarP(&cmd.flags.deployment, "deployment", "d", "", "The deployment to delete (You can specify multiple deployments comma-separated, e.g. devspace-default,devspace-database etc.)")
	cobraCmd.Flags().StringVar(&cmd.flags.config, "config", configutil.ConfigPath, "The devspace config file to load (default: '.devspace/config.yaml')")

	rootCmd.AddCommand(cobraCmd)
}

// Run executes the purge command logic
func (cmd *PurgeCmd) Run(cobraCmd *cobra.Command, args []string) {
	if configutil.ConfigPath != cmd.flags.config {
		configutil.ConfigPath = cmd.flags.config
	}

	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find any devspace configuration. Please run `devspace init`")
	}

	log.StartFileLogging()

	// Configure cloud provider
	err = cloud.Configure(log.GetInstance())
	if err != nil {
		log.Fatalf("Unable to configure cloud provider: %v", err)
	}

	kubectl, err := kubectl.NewClient()
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %s", err.Error())
	}

	deployments := []string{}
	if cmd.flags.deployment != "" {
		deployments = strings.Split(cmd.flags.deployment, ",")
		for index := range deployments {
			deployments[index] = strings.TrimSpace(deployments[index])
		}
	}

	deleteDevSpace(kubectl, deployments)
}

func deleteDevSpace(kubectl *kubernetes.Clientset, deployments []string) {
	config := configutil.GetConfig()
	if deployments != nil && len(deployments) == 0 {
		deployments = nil
	}

	if config.Deployments != nil {
		// Reverse them
		for i := len(*config.Deployments) - 1; i >= 0; i-- {
			deployConfig := (*config.Deployments)[i]

			// Check if we should skip deleting deployment
			if deployments != nil {
				found := false

				for _, value := range deployments {
					if value == *deployConfig.Name {
						found = true
						break
					}
				}

				if found == false {
					continue
				}
			}

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

			log.StartWait("Deleting deployment " + *deployConfig.Name)
			err = deployClient.Delete()
			log.StopWait()
			if err != nil {
				log.Warnf("Error deleting deployment %s: %v", *deployConfig.Name, err)
			}

			log.Donef("Successfully deleted deployment %s", *deployConfig.Name)
		}
	}
}
