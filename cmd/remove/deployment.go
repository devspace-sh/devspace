package remove

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/configure"
	deployUtil "github.com/devspace-cloud/devspace/pkg/devspace/deploy/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type deploymentCmd struct {
	*flags.GlobalFlags

	RemoveAll bool
}

func newDeploymentCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &deploymentCmd{GlobalFlags: globalFlags}

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
		RunE: cmd.RunRemoveDeployment,
	}

	deploymentCmd.Flags().BoolVar(&cmd.RemoveAll, "all", false, "Remove all deployments")

	return deploymentCmd
}

// RunRemoveDeployment executes the specified deployment
func (cmd *deploymentCmd) RunRemoveDeployment(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot(log.GetInstance())
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	// Load base config
	config, err := configutil.GetBaseConfig(cmd.ToConfigOptions())
	if err != nil {
		return err
	}

	shouldPurgeDeployment, err := survey.Question(&survey.QuestionOptions{
		Question:     "Do you want to delete all deployment resources deployed?",
		DefaultValue: "yes",
		Options: []string{
			"yes",
			"no",
		},
	}, log.GetInstance())
	if err != nil {
		return err
	}
	if shouldPurgeDeployment == "yes" {
		client, err := kubectl.NewDefaultClient()
		if err != nil {
			return errors.Errorf("Unable to create new kubectl client: %v", err)
		}

		deployments := []string{}
		if cmd.RemoveAll == false {
			deployments = []string{name}
		}

		generatedConfig, err := generated.LoadConfig("")
		if err != nil {
			log.Errorf("Error loading generated.yaml: %v", err)
			return nil
		}

		deployUtil.PurgeDeployments(config, generatedConfig.GetActive(), client, deployments, log.GetInstance())

		err = generated.SaveConfig(generatedConfig)
		if err != nil {
			log.Errorf("Error saving generated.yaml: %v", err)
		}
	}

	found, err := configure.RemoveDeployment(config, cmd.RemoveAll, name)
	if err != nil {
		return err
	}

	if found {
		if cmd.RemoveAll {
			log.Donef("Successfully removed all deployments")
		} else {
			log.Donef("Successfully removed deployment %s", name)
		}
	} else {
		if cmd.RemoveAll {
			log.Warnf("Couldn't find any deployment")
		} else {
			log.Warnf("Couldn't find deployment %s", name)
		}
	}

	return nil
}
