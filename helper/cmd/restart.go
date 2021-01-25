package cmd

import (
	"github.com/loft-sh/devspace/helper/util"
	"github.com/spf13/cobra"
)

// RestartCmd holds the cmd flags
type RestartCmd struct{}

// NewRestartCmd creates a new restart command
func NewRestartCmd() *cobra.Command {
	cmd := &RestartCmd{}
	restartCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restarts the container if the restart helper is present",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}

	return restartCmd
}

// Run runs the command logic
func (cmd *RestartCmd) Run(cobraCmd *cobra.Command, args []string) error {
	return util.NewContainerRestarter().RestartContainer()
}
