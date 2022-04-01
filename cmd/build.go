package cmd

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewBuildCmd creates a new devspace build command
func NewBuildCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &RunPipelineCmd{
		GlobalFlags: globalFlags,
		Pipeline:    "build",
		ForceBuild:  true,
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
			return cmd.Run(cobraCmd, args, f, "build", "buildCommand")
		},
	}

	cmd.AddFlags(buildCmd)
	return buildCmd
}
