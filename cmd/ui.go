package cmd

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
)

// UICmd holds the open cmd flags
type UICmd struct{}

// NewUICmd creates a new ui command
func NewUICmd() *cobra.Command {
	cmd := &UICmd{}

	uiCmd := &cobra.Command{
		Use:   "ui",
		Short: "Opens the management ui in the browser",
		Long: `
#######################################################
##################### devspace ui #####################
#######################################################
Opens the management ui in the browser
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: cmd.RunUI,
	}

	return uiCmd
}

// RunUI executes the functionality "devspace ui"
func (cmd *UICmd) RunUI(cobraCmd *cobra.Command, args []string) error {
	// Get provider
	provider, err := cloud.GetProvider(nil, log.GetInstance())
	if err != nil {
		return err
	}

	open.Start(provider.Host)
	log.Donef("Successfully opened %s", provider.Host)

	return nil
}
