package remove

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/configure"
	deployUtil "github.com/devspace-cloud/devspace/pkg/devspace/deploy/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/spf13/cobra"
)

type deploymentCmd struct {
	RemoveAll bool
}

func newDeploymentCmd() *cobra.Command {
	cmd := &deploymentCmd{}

	deploymentCmd := &cobra.Command{
		Use:   "deployment [deployment-name]",
		Short: "Removes one or all deployments from devspace configuration",
		Long: `
#######################################################
############ devspace remove deployment ###############
#######################################################
Removes one or all deployments from the devspace
configuration (If you want to delete the deployed 
resources, run 'devspace purge -d deployment_name'):

devspace remove deployment devspace-default
devspace remove deployment --all
#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		Run:  cmd.RunRemoveDeployment,
	}

	deploymentCmd.Flags().BoolVar(&cmd.RemoveAll, "all", false, "Remove all deployments")

	return deploymentCmd
}

// RunRemoveDeployment executes the specified deployment
func (cmd *deploymentCmd) RunRemoveDeployment(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	// Load base config
	config := configutil.GetBaseConfig()

	shouldPurgeDeployment := survey.Question(&survey.QuestionOptions{
		Question:     "Do you want to delete all deployment resources deployed?",
		DefaultValue: "yes",
		Options: []string{
			"yes",
			"no",
		},
	}) == "yes"
	if shouldPurgeDeployment {
		kubectl, err := kubectl.NewClient(config)
		if err != nil {
			log.Fatalf("Unable to create new kubectl client: %v", err)
		}

		deployments := []string{}
		if cmd.RemoveAll == false {
			deployments = []string{args[0]}
		}

		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			log.Errorf("Error loading generated.yaml: %v", err)
			return
		}

		deployUtil.PurgeDeployments(config, generatedConfig.GetActive(), kubectl, deployments, log.GetInstance())

		err = generated.SaveConfig(generatedConfig)
		if err != nil {
			log.Errorf("Error saving generated.yaml: %v", err)
		}
	}

	found, err := configure.RemoveDeployment(cmd.RemoveAll, name)
	if err != nil {
		log.Fatal(err)
	}

	if found {
		if cmd.RemoveAll {
			log.Donef("Successfully removed all deployments")
		} else {
			log.Donef("Successfully removed deployment %s", args[0])
		}
	} else {
		if cmd.RemoveAll {
			log.Warnf("Couldn't find any deployment")
		} else {
			log.Warnf("Couldn't find deployment %s", args[0])
		}
	}
}
