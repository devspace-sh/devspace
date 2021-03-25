package remove

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/loft-sh/devspace/pkg/util/survey"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type deploymentCmd struct {
	*flags.GlobalFlags

	RemoveAll bool
}

func newDeploymentCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunRemoveDeployment(f, cobraCmd, args)
		}}

	deploymentCmd.Flags().BoolVar(&cmd.RemoveAll, "all", false, "Remove all deployments")

	return deploymentCmd
}

// RunRemoveDeployment executes the specified deployment
func (cmd *deploymentCmd) RunRemoveDeployment(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	log := f.GetLog()
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	log.Warn("This command is deprecated and will be removed in a future DevSpace version. Please modify the devspace.yaml directly instead")
	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	// Load base config
	configWrapper, err := configLoader.Load(cmd.ToConfigOptions(), log)
	if err != nil {
		return err
	}

	config := configWrapper.Config()
	shouldPurgeDeployment, err := log.Question(&survey.QuestionOptions{
		Question:     "Do you want to delete all deployment resources deployed?",
		DefaultValue: "yes",
		Options: []string{
			"yes",
			"no",
		},
	})
	if err != nil {
		return err
	}
	if shouldPurgeDeployment == "yes" {
		client, err := f.NewKubeDefaultClient()
		if err != nil {
			return errors.Errorf("Unable to create new kubectl client: %v", err)
		}

		deployments := []string{}
		if cmd.RemoveAll == false {
			deployments = []string{name}
		}

		generatedConfig := configWrapper.Generated()
		err = deploy.NewController(config, generatedConfig.GetActive(), client).Purge(deployments, log)
		if err != nil {
			log.Errorf("Error purging deployments: %v", err)
		}

		err = configLoader.SaveGenerated(generatedConfig)
		if err != nil {
			log.Errorf("Error saving generated.yaml: %v", err)
		}
	}

	configureManager := f.NewConfigureManager(config, log)
	found, err := configureManager.RemoveDeployment(cmd.RemoveAll, name)
	if err != nil {
		return err
	}

	err = configLoader.Save(config)
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
