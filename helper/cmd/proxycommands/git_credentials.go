package proxycommands

import (
	"github.com/spf13/cobra"
)

// GitCredentials holds the cmd flags
type GitCredentials struct{}

// NewGitCredentialsCmd creates a new ssh command
func NewGitCredentialsCmd() *cobra.Command {
	cmd := &GitCredentials{}
	runCmd := &cobra.Command{
		Use:                "git-credentials",
		Short:              "Retrieves git credentials from local",
		DisableFlagParsing: true,
		RunE:               cmd.Run,
	}
	return runCmd
}

// Run runs the command logic
func (cmd *GitCredentials) Run(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	} else if args[0] != "get" {
		return nil
	}

	return runProxyCommand([]string{"git-credentials", "credential", "fill"})
}
