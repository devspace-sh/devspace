package cmd

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewBuildCmd creates a new devspace build command
func NewBuildCmd(f factory.Factory, globalFlags *flags.GlobalFlags, rawConfig *RawConfig) *cobra.Command {
	cmd := &RunPipelineCmd{
		GlobalFlags:             globalFlags,
		Pipeline:                "build",
		ForceBuild:              true,
		SkipPushLocalKubernetes: true,
	}

	var pipeline *latest.Pipeline
	if rawConfig != nil && rawConfig.Config != nil && rawConfig.Config.Pipelines != nil {
		pipeline = rawConfig.Config.Pipelines["build"]
	}
	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Builds all defined images and pushes them",
		Long: `
#######################################################
################## devspace build #####################
#######################################################
Builds all defined images and pushes them
#######################################################`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd, args, f, "buildCommand")
		},
	}
	cmd.AddPipelineFlags(f, buildCmd, pipeline)
	return buildCmd
}
