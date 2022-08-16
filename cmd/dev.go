package cmd

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewDevCmd creates a new devspace dev command
func NewDevCmd(f factory.Factory, globalFlags *flags.GlobalFlags, rawConfig *RawConfig) *cobra.Command {
	cmd := &RunPipelineCmd{
		GlobalFlags:             globalFlags,
		SkipPushLocalKubernetes: true,
		Pipeline:                "dev",
	}

	var pipeline *latest.Pipeline
	if rawConfig != nil && rawConfig.Config != nil && rawConfig.Config.Pipelines != nil {
		pipeline = rawConfig.Config.Pipelines["dev"]
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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd, args, f, "devCommand")
		},
	}
	cmd.AddPipelineFlags(f, devCmd, pipeline)
	return devCmd
}
