package restore

import (
	"fmt"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type varsCmd struct {
	*flags.GlobalFlags

	SecretName string
}

func newVarsCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &varsCmd{
		GlobalFlags: globalFlags,
	}

	varsCmd := &cobra.Command{
		Use:   "vars",
		Short: "Restores variable values from kubernetes",
		Long: `
#######################################################
############### devspace restore vars #################
#######################################################
Restores devspace config variable values from a kubernetes
secret. 

Examples:
devspace restore vars
devspace restore vars --namespace test 
devspace restore vars --vars-secret my-secret
#######################################################
	`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		}}

	varsCmd.Flags().StringVar(&cmd.SecretName, "vars-secret", "devspace-vars", "The secret to restore the variables from")
	return varsCmd
}

// RunSetVar executes the set var command logic
func (cmd *varsCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	logger := f.GetLog()
	configLoader := f.NewConfigLoader("")
	configExists, err := configLoader.SetDevSpaceRoot(logger)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Load generated config
	generatedConfig, err := configLoader.LoadGenerated(nil)
	if err != nil {
		return err
	}

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, logger)
	if err != nil {
		return err
	}

	// Get kubectl client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return errors.Wrap(err, "new kube client")
	}

	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, logger)
	if err != nil {
		return err
	}

	vars, restored, err := loader.RestoreVarsFromSecret(client, cmd.SecretName)
	if err != nil {
		return err
	} else if restored == false {
		logger.Infof("No saved variables found in namespace %s", client.Namespace())
		return nil
	}

	// exchange vars
	generatedConfig.Vars = vars

	// Make sure the vars are also saved to file
	err = configLoader.SaveGenerated(generatedConfig)
	if err != nil {
		return fmt.Errorf("error saving generated.yaml: %v", err)
	}

	logger.Donef("Successfully restored vars from secret %s/%s", client.Namespace(), cmd.SecretName)
	return nil
}
