package cmd

import (
	"github.com/covexo/devspace/pkg/devspace/cloud"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// LoginCmd holds the login cmd flags
type LoginCmd struct {
	Token    string
	Provider string
}

// NewLoginCmd creates a new login command
func NewLoginCmd() *cobra.Command {
	cmd := &LoginCmd{}

	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Log into devspace cloud",
		Long: `
	#######################################################
	################### devspace login ####################
	#######################################################
	If no --token is supplied the browser will be opened 
	and the login page is shown

	Example:
	devspace login
	devspace login --token 123456789
	#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunLogin,
	}

	loginCmd.Flags().StringVar(&cmd.Token, "token", "", "Token to use for login")
	loginCmd.Flags().StringVar(&cmd.Provider, "provider", cloud.DevSpaceCloudProviderName, "Cloud provider to use")

	return loginCmd
}

// RunLogin executes the functionality devspace login
func (cmd *LoginCmd) RunLogin(cobraCmd *cobra.Command, args []string) {
	providerConfig, err := cloud.ParseCloudConfig()
	if err != nil {
		log.Fatal(err)
	}

	if cmd.Token != "" {
		err = cloud.ReLogin(providerConfig, cmd.Provider, &cmd.Token, log.GetInstance())
		if err != nil {
			log.Fatalf("Error logging in: %v", err)
		}
	} else {
		err = cloud.ReLogin(providerConfig, cmd.Provider, nil, log.GetInstance())
		if err != nil {
			log.Fatalf("Error logging in: %v", err)
		}
	}

	log.Infof("Successful logged into %s", cmd.Provider)
}
