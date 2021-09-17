package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version string

// NewVersionCmd creates a new version command
func NewVersionCmd() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Prints the cli version",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			if version == "" {
				version = "latest"
			}

			fmt.Fprint(os.Stdout, version)
			return nil
		},
	}

	return versionCmd
}
