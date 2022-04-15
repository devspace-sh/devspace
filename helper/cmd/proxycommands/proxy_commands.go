package proxycommands

import (
	"github.com/spf13/cobra"
)

// NewProxyCommands creates a new cobra command
func NewProxyCommands() *cobra.Command {
	reverseCommandsCmd := &cobra.Command{
		Use:   "proxy-commands",
		Short: "Execute Commands on a remote environment",
		Args:  cobra.NoArgs,
	}

	reverseCommandsCmd.AddCommand(NewConfigureCmd())
	reverseCommandsCmd.AddCommand(NewRunCmd())
	reverseCommandsCmd.AddCommand(NewGitCredentialsCmd())
	return reverseCommandsCmd
}
