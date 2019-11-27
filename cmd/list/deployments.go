package list

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/resume"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/deployer"
	deployHelm "github.com/devspace-cloud/devspace/pkg/devspace/deploy/deployer/helm"
	deployKubectl "github.com/devspace-cloud/devspace/pkg/devspace/deploy/deployer/kubectl"
	helmtypes "github.com/devspace-cloud/devspace/pkg/devspace/helm/types"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	logpkg "github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type deploymentsCmd struct {
	*flags.GlobalFlags
}

func newDeploymentsCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &deploymentsCmd{GlobalFlags: globalFlags}

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
		RunE: cmd.RunDeploymentsStatus,
	}
}

// RunDeploymentsStatus executes the devspace status deployments command logic
func (cmd *deploymentsCmd) RunDeploymentsStatus(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	log := logpkg.GetInstance()
	configLoader := loader.NewConfigLoader(cmd.ToConfigOptions(), log)
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	var values [][]string
	var headerValues = []string{
		"NAME",
		"TYPE",
		"DEPLOY",
		"STATUS",
	}

	// Load generated
	generatedConfig, err := configLoader.Generated()
	if err != nil {
		return err
	}

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, log)
	if err != nil {
		return err
	}

	// Create new kube client
	client, err := kubectl.NewClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return err
	}

	// Show warning if the old kube context was different
	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, log)
	if err != nil {
		return err
	}

	// Get config with adjusted cluster config
	config, err := configLoader.Load()
	if err != nil {
		return err
	}

	// Signal that we are working on the space if there is any
	resumer := resume.NewSpaceResumer(client, log)
	err = resumer.ResumeSpace(true)
	if err != nil {
		return err
	}

	if config.Deployments != nil {
		helmV2Clients := map[string]helmtypes.Client{}

		for _, deployConfig := range config.Deployments {
			var deployClient deployer.Interface

			// Delete kubectl engine
			if deployConfig.Kubectl != nil {
				deployClient, err = deployKubectl.New(config, client, deployConfig, log)
				if err != nil {
					log.Warnf("Unable to create kubectl deploy config for %s: %v", deployConfig.Name, err)
					continue
				}
			} else if deployConfig.Helm != nil {
				helmClient, err := deploy.GetCachedHelmClient(config, deployConfig, client, helmV2Clients, log)
				if err != nil {
					log.Warnf("Unable to create helm deploy config for %s: %v", deployConfig.Name, err)
					continue
				}

				deployClient, err = deployHelm.New(config, helmClient, client, deployConfig, log)
				if err != nil {
					log.Warnf("Unable to create helm deploy config for %s: %v", deployConfig.Name, err)
					continue
				}
			} else {
				log.Warnf("No deployment method defined for deployment %s", deployConfig.Name)
				continue
			}

			status, err := deployClient.Status()
			if err != nil {
				log.Warnf("Error retrieving status for deployment %s: %v", deployConfig.Name, err)
				continue
			}

			values = append(values, []string{
				status.Name,
				status.Type,
				status.Target,
				status.Status,
			})
		}
	}

	logpkg.PrintTable(log, headerValues, values)
	return nil
}
