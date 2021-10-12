package cmd

import (
	"os"

	"github.com/loft-sh/devspace/helper/tunnel"
	"github.com/spf13/cobra"
)

// TunnelCmd holds the tunnel cmd flags
type TunnelCmd struct{}

// NewTunnelCmd creates a new tunnel command
func NewTunnelCmd() *cobra.Command {
	cmd := &TunnelCmd{}
	tunnelCmd := &cobra.Command{
		Use:   "tunnel",
		Short: "Starts a new tunnel for reverse port forwarding",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}

	return tunnelCmd
}

// Run runs the command logic
func (cmd *TunnelCmd) Run(cobraCmd *cobra.Command, args []string) error {
	return tunnel.StartTunnelServer(os.Stdin, os.Stdout, true, true)
}
