package cmd

import (
	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// LoginCmd holds the login cmd flags
type LoginCmd struct {
	Key      string
	Provider string
}

// NewLoginCmd creates a new login command
func NewLoginCmd(f factory.Factory) *cobra.Command {
	cmd := &LoginCmd{}

	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Log into DevSpace Cloud",
		Long: `
#######################################################
################### devspace login ####################
#######################################################
If no --key is supplied the browser will be opened 
and the login page is shown

Example:
devspace login
devspace login --key myaccesskey
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunLogin(f, cobraCmd, args)
		},
	}

	loginCmd.Flags().StringVar(&cmd.Key, "key", "", "Access key to use")
	loginCmd.Flags().StringVar(&cmd.Provider, "provider", "", "Provider to use")

	return loginCmd
}

// RunLogin executes the functionality devspace login
func (cmd *LoginCmd) RunLogin(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	log := f.GetLog()
	kubeLoader := f.NewKubeConfigLoader()
	loader := f.NewCloudConfigLoader()
	providerConfig, err := loader.Load()
	if err != nil {
		return err
	}

	providerName := cloudconfig.DevSpaceCloudProviderName
	if providerConfig.Default != "" {
		providerName = providerConfig.Default
	}
	if cmd.Provider != "" {
		providerName = cmd.Provider
	}

	if cmd.Key != "" {
		_, err = f.GetProviderWithOptions(providerName, cmd.Key, true, loader, kubeLoader, log)
		if err != nil {
			return err
		}
	} else {
		_, err = f.GetProviderWithOptions(providerName, "", true, loader, kubeLoader, log)
		if err != nil {
			return err
		}
	}

	log.Infof("Successful logged into %s", providerName)
	return nil
}
