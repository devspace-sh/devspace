package cmd

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/factory"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/spf13/cobra"
)

// NewPurgeCmd creates a new purge command
func NewPurgeCmd(f factory.Factory, globalFlags *flags.GlobalFlags, rawConfig *RawConfig) *cobra.Command {
	cmd := &RunPipelineCmd{
		GlobalFlags:             globalFlags,
		Pipeline:                "purge",
		SkipPushLocalKubernetes: true,
	}

	var pipeline *latest.Pipeline
	if rawConfig != nil && rawConfig.Config != nil && rawConfig.Config.Pipelines != nil {
		pipeline = rawConfig.Config.Pipelines["purge"]
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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd, args, f, "purgeCommand")
		},
	}
	cmd.AddPipelineFlags(f, purgeCmd, pipeline)
	return purgeCmd
}
