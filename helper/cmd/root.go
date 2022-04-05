package cmd

import (
	"fmt"
	"os"

	"github.com/loft-sh/devspace/helper/cmd/proxycommands"

	"github.com/loft-sh/devspace/helper/cmd/sync"
	"github.com/spf13/cobra"
)

// NewRootCmd returns a new root command
func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "devspacehelper",
		Short: "DevSpace Utility CLI",
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// build the root command
	rootCmd := BuildRoot()

	// execute command
	err := rootCmd.Execute()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}

// BuildRoot creates a new root command from the
func BuildRoot() *cobra.Command {
	rootCmd := NewRootCmd()

	rootCmd.AddCommand(NewRestartCmd())
	rootCmd.AddCommand(NewVersionCmd())
	rootCmd.AddCommand(NewTunnelCmd())
	rootCmd.AddCommand(NewSSHCmd())
	rootCmd.AddCommand(sync.NewSyncCmd())
	rootCmd.AddCommand(proxycommands.NewProxyCommands())
	return rootCmd
}
