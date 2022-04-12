package cmd

import (
	"github.com/loft-sh/devspace/pkg/util/factory"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/spf13/cobra"
)

// NewPurgeCmd creates a new purge command
func NewPurgeCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &RunPipelineCmd{
		GlobalFlags:             globalFlags,
		Pipeline:                "purge",
		SkipPushLocalKubernetes: true,
	}
	purgeCmd := &cobra.Command{
		Use:   "purge",
		Short: "Deletes deployed resources",
		Long: `
#######################################################
################### devspace purge ####################
#######################################################
Deletes the deployed kubernetes resources:

devspace purge
#######################################################`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd, args, f, "purge", "purgeCommand")
		},
	}

	cmd.AddFlags(purgeCmd)
	return purgeCmd
}
