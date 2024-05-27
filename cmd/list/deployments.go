package list

import (
	"context"

	"github.com/loft-sh/devspace/cmd/flags"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer"
	deployHelm "github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/helm"
	deployKubectl "github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/kubectl"
	deployTanka "github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/tanka"
	"github.com/loft-sh/devspace/pkg/devspace/helm"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
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
Lists the status of all deployments
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
	configOptions := cmd.ToConfigOptions()
	configLoader, err := f.NewConfigLoader(cmd.ConfigPath)
	if err != nil {
		return err
	}
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

	// Create new kube client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace)
	if err != nil {
		return err
	}

	// Load generated
	localCache, err := configLoader.LoadLocalCache()
	if err != nil {
		return err
	}

	// If the current kube context or namespace is different from old,
	// show warnings and reset kube client if necessary
	client, err = kubectl.CheckKubeContext(client, localCache, cmd.NoWarn, cmd.SwitchContext, false, logger)
	if err != nil {
		return err
	}

	// Get config with adjusted cluster config
	configInterface, err := configLoader.LoadWithCache(context.Background(), localCache, client, configOptions, logger)
	if err != nil {
		return err
	}

	// Create conext
	ctx := devspacecontext.NewContext(context.Background(), configInterface.Variables(), logger).
		WithConfig(configInterface).
		WithKubeClient(client)

	// Resolve dependencies
	dependencies, err := f.NewDependencyManager(ctx, configOptions).ResolveAll(ctx, dependency.ResolveOptions{})
	if err != nil {
		return err
	}
	ctx = ctx.WithDependencies(dependencies)

	if ctx.Config().Config().Deployments != nil {
		for _, deployConfig := range ctx.Config().Config().Deployments {
			var deployClient deployer.Interface

			// Delete kubectl engine
			if deployConfig.Kubectl != nil {
				deployClient, err = deployKubectl.New(ctx, deployConfig)
				if err != nil {
					logger.Warnf("Unable to create kubectl deploy config for %s: %v", deployConfig.Name, err)
					continue
				}
			} else if deployConfig.Helm != nil {
				helmClient, err := helm.NewClient(logger)
				if err != nil {
					logger.Warnf("Unable to create helm deploy config for %s: %v", deployConfig.Name, err)
					continue
				}

				deployClient, err = deployHelm.New(helmClient, deployConfig)
				if err != nil {
					logger.Warnf("Unable to create helm deploy config for %s: %v", deployConfig.Name, err)
					continue
				}
			} else if deployConfig.Tanka != nil {
				deployClient, err = deployTanka.New(ctx, deployConfig)
				if err != nil {
					logger.Warnf("Unable to create tanka deploy config for %s: %v", deployConfig.Name, err)
					continue
				}
			} else {
				logger.Warnf("No deployment method defined for deployment %s", deployConfig.Name)
				continue
			}

			status, err := deployClient.Status(ctx)
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
