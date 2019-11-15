package cmd

import (
	"strings"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/dependency"
	deploy "github.com/devspace-cloud/devspace/pkg/devspace/deploy/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

// PurgeCmd holds the required data for the purge cmd
type PurgeCmd struct {
	*flags.GlobalFlags

	Deployments             string
	AllowCyclicDependencies bool
	VerboseDependencies     bool
	PurgeDependencies       bool
}

// NewPurgeCmd creates a new purge command
func NewPurgeCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &PurgeCmd{GlobalFlags: globalFlags}

	purgeCmd := &cobra.Command{
		Use:   "purge",
		Short: "Delete deployed resources",
		Long: `
#######################################################
################### devspace purge ####################
#######################################################
Deletes the deployed kuberenetes resources:

devspace purge
devspace purge --dependencies
devspace purge -d my-deployment
#######################################################`,
		Args: cobra.NoArgs,
		RunE: cmd.Run,
	}

	purgeCmd.Flags().StringVarP(&cmd.Deployments, "deployments", "d", "", "The deployment to delete (You can specify multiple deployments comma-separated, e.g. devspace-default,devspace-database etc.)")
	purgeCmd.Flags().BoolVar(&cmd.AllowCyclicDependencies, "allow-cyclic", false, "When enabled allows cyclic dependencies")
	purgeCmd.Flags().BoolVar(&cmd.PurgeDependencies, "dependencies", false, "When enabled purges the dependencies as well")
	purgeCmd.Flags().BoolVar(&cmd.VerboseDependencies, "verbose-dependencies", false, "Builds the dependencies verbosely")

	return purgeCmd
}

// Run executes the purge command logic
func (cmd *PurgeCmd) Run(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot(log.GetInstance())
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	log.StartFileLogging()

	// Get config with adjusted cluster config
	generatedConfig, err := generated.LoadConfig(cmd.Profile)
	if err != nil {
		return err
	}

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, log.GetInstance())
	if err != nil {
		return err
	}

	client, err := kubectl.NewClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return errors.Wrap(err, "create kube client")
	}

	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, log.GetInstance())
	if err != nil {
		return err
	}

	// Signal that we are working on the space if there is any
	err = cloud.ResumeSpace(client, true, log.GetInstance())
	if err != nil {
		return err
	}

	// Get config with adjusted cluster config
	configOptions := cmd.ToConfigOptions()
	config, err := configutil.GetConfig(configOptions)
	if err != nil {
		return err
	}

	deployments := []string{}
	if cmd.Deployments != "" {
		deployments = strings.Split(cmd.Deployments, ",")
		for index := range deployments {
			deployments[index] = strings.TrimSpace(deployments[index])
		}
	}

	// Purge deployments
	deploy.PurgeDeployments(config, generatedConfig.GetActive(), client, deployments, log.GetInstance())

	// Purge dependencies
	if cmd.PurgeDependencies {

		// Create Dependencymanager
		manager, err := dependency.NewManager(config, generatedConfig, client, cmd.AllowCyclicDependencies, cmd.ToConfigOptions(), log.GetInstance())
		if err != nil {
			return errors.Wrap(err, "new manager")
		}

		err = manager.PurgeAll(cmd.VerboseDependencies)
		if err != nil {
			log.Errorf("Error purging dependencies: %v", err)
		}
	}

	err = generated.SaveConfig(generatedConfig)
	if err != nil {
		log.Errorf("Error saving generated.yaml: %v", err)
	}

	return nil
}
