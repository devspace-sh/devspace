package list

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy"
	deployComponent "github.com/devspace-cloud/devspace/pkg/devspace/deploy/component"
	deployHelm "github.com/devspace-cloud/devspace/pkg/devspace/deploy/helm"
	deployKubectl "github.com/devspace-cloud/devspace/pkg/devspace/deploy/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"

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
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	var values [][]string
	var headerValues = []string{
		"NAME",
		"TYPE",
		"DEPLOY",
		"STATUS",
	}

	// Load generated
	generatedConfig, err := generated.LoadConfig(cmd.Profile)
	if err != nil {
		return err
	}

	// Create new kube client
	client, err := kubectl.NewClientFromContext(cmd.KubeContext, cmd.Namespace, false)
	if err != nil {
		return err
	}

	// Show warning if the old kube context was different
	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, log.GetInstance())
	if err != nil {
		return err
	}

	// Get config with adjusted cluster config
	config, err := configutil.GetConfig(cmd.KubeContext, cmd.Profile)
	if err != nil {
		return err
	}

	// Signal that we are working on the space if there is any
	err = cloud.ResumeSpace(client, true, log.GetInstance())
	if err != nil {
		return err
	}

	if config.Deployments != nil {
		for _, deployConfig := range config.Deployments {
			var deployClient deploy.Interface

			// Delete kubectl engine
			if deployConfig.Kubectl != nil {
				deployClient, err = deployKubectl.New(config, client, deployConfig, log.GetInstance())
				if err != nil {
					log.Warnf("Unable to create kubectl deploy config for %s: %v", deployConfig.Name, err)
					continue
				}
			} else if deployConfig.Helm != nil {
				deployClient, err = deployHelm.New(config, client, deployConfig, log.GetInstance())
				if err != nil {
					log.Warnf("Unable to create helm deploy config for %s: %v", deployConfig.Name, err)
					continue
				}
			} else if deployConfig.Component != nil {
				deployClient, err = deployComponent.New(config, client, deployConfig, log.GetInstance())
				if err != nil {
					log.Warnf("Unable to create component deploy config for %s: %v", deployConfig.Name, err)
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

	log.PrintTable(log.GetInstance(), headerValues, values)
	return nil
}
