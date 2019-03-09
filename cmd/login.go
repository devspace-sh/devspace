package cmd

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// LoginCmd holds the login cmd flags
type LoginCmd struct {
	Token string
}

// NewLoginCmd creates a new login command
func NewLoginCmd() *cobra.Command {
	cmd := &LoginCmd{}

	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Log into DevSpace Cloud",
		Long: `
#######################################################
################### devspace login ####################
#######################################################
If no --token is supplied the browser will be opened 
and the login page is shown

Example:
devspace login
devspace login my.custom.cloud
devspace login --token 123456789
#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		Run:  cmd.RunLogin,
	}

	loginCmd.Flags().StringVar(&cmd.Token, "token", "", "Token to use for login")

	return loginCmd
}

// RunLogin executes the functionality devspace login
func (cmd *LoginCmd) RunLogin(cobraCmd *cobra.Command, args []string) {
	providerConfig, err := cloud.ParseCloudConfig()
	if err != nil {
		log.Fatal(err)
	}

	providerName := cloud.DevSpaceCloudProviderName
	if len(args) > 0 {
		providerName = args[0]
	}

	if cmd.Token != "" {
		err = cloud.ReLogin(providerConfig, providerName, &cmd.Token, log.GetInstance())
		if err != nil {
			log.Fatalf("Error logging in: %v", err)
		}
	} else {
		err = cloud.ReLogin(providerConfig, providerName, nil, log.GetInstance())
		if err != nil {
			log.Fatalf("Error logging in: %v", err)
		}
	}

	log.Infof("Successful logged into %s", providerName)
}
