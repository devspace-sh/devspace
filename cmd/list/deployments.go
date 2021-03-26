package list

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer"
	deployHelm "github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/helm"
	deployKubectl "github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/kubectl"
	helmtypes "github.com/loft-sh/devspace/pkg/devspace/helm/types"
	"github.com/loft-sh/devspace/pkg/util/factory"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type deploymentsCmd struct {
	*flags.GlobalFlags
}

func newDeploymentsCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunDeploymentsStatus(f, cobraCmd, args)
		}}
}

// RunDeploymentsStatus executes the devspace status deployments command logic
func (cmd *deploymentsCmd) RunDeploymentsStatus(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	logger := f.GetLog()
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(logger)
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
	generatedConfig, err := configLoader.LoadGenerated(cmd.ToConfigOptions())
	if err != nil {
		return err
	}

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, logger)
	if err != nil {
		return err
	}

	// Create new kube client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return err
	}

	// Show warning if the old kube context was different
	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, logger)
	if err != nil {
		return err
	}

	// Get config with adjusted cluster config
	configInterface, err := configLoader.Load(cmd.ToConfigOptions(), logger)
	if err != nil {
		return err
	}

	config := configInterface.Config()
	if config.Deployments != nil {
		helmV2Clients := map[string]helmtypes.Client{}

		for _, deployConfig := range config.Deployments {
			var deployClient deployer.Interface

			// Delete kubectl engine
			if deployConfig.Kubectl != nil {
				deployClient, err = deployKubectl.New(config, client, deployConfig, logger)
				if err != nil {
					logger.Warnf("Unable to create kubectl deploy config for %s: %v", deployConfig.Name, err)
					continue
				}
			} else if deployConfig.Helm != nil {
				helmClient, err := deploy.GetCachedHelmClient(config, deployConfig, client, helmV2Clients, false, logger)
				if err != nil {
					logger.Warnf("Unable to create helm deploy config for %s: %v", deployConfig.Name, err)
					continue
				}

				deployClient, err = deployHelm.New(config, helmClient, client, deployConfig, logger)
				if err != nil {
					logger.Warnf("Unable to create helm deploy config for %s: %v", deployConfig.Name, err)
					continue
				}
			} else {
				logger.Warnf("No deployment method defined for deployment %s", deployConfig.Name)
				continue
			}

			status, err := deployClient.Status()
			if err != nil {
				logger.Warnf("Error retrieving status for deployment %s: %v", deployConfig.Name, err)
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

	logpkg.PrintTable(logger, headerValues, values)
	return nil
}
