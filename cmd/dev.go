package cmd

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewDevCmd creates a new devspace dev command
func NewDevCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &RunPipelineCmd{
		GlobalFlags:             globalFlags,
		SkipPushLocalKubernetes: true,
		Pipeline:                "dev",
	}
	devCmd := &cobra.Command{
		Use:   "dev",
		Short: "Starts the development mode",
		Long: `
#######################################################
################### devspace dev ######################
#######################################################
Starts your project in development mode
#######################################################`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd, args, f, "dev", "devCommand")
		},
	}

	cmd.AddFlags(devCmd)
	return devCmd
}
