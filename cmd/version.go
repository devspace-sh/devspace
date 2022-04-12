package cmd

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"
	"github.com/spf13/cobra"
)

// NewVersionCmd returns the cobra command that outputs version
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Args:  cobra.NoArgs,
		Short: "Prints version of devspace",
		Run: func(cobraCmd *cobra.Command, args []string) {
			fmt.Println("DevSpace version : " + upgrade.GetVersion())
		},
	}
}
