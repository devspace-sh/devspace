package reverse_commands

import (
	"github.com/spf13/cobra"
)

// NewReverseCommands creates a new cobra command
func NewReverseCommands() *cobra.Command {
	reverseCommandsCmd := &cobra.Command{
		Use:   "reverse-commands",
		Short: "Execute Commands on a remote environment",
		Args:  cobra.NoArgs,
	}

	reverseCommandsCmd.AddCommand(NewConfigureCmd())
	reverseCommandsCmd.AddCommand(NewRunCmd())
	return reverseCommandsCmd
}
