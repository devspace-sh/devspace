package cmd

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"os"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewRenderCmd creates a new devspace render command
func NewRenderCmd(f factory.Factory, globalFlags *flags.GlobalFlags, rawConfig *RawConfig) *cobra.Command {
	cmd := &RunPipelineCmd{
		GlobalFlags:             globalFlags,
		SkipPushLocalKubernetes: true,
		Pipeline:                "deploy",
		Render:                  true,
		RenderWriter:            os.Stdout,
	}

	var pipeline *latest.Pipeline
	if rawConfig != nil && rawConfig.Config != nil && rawConfig.Config.Pipelines != nil {
		pipeline = rawConfig.Config.Pipelines["deploy"]
	}
	renderCmd := &cobra.Command{
		Use:   "render",
		Short: "Builds all defined images and shows the yamls that would be deployed",
		Long: `
#######################################################
################## devspace render #####################
#######################################################
Builds all defined images and shows the yamls that would
be deployed via helm and kubectl, but skips actual 
deployment.
#######################################################`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			f.GetLog().Warnf("This command is deprecated, please use 'devspace deploy --render' instead")
			return cmd.Run(cobraCmd, args, f, "renderCommand")
		},
	}
	cmd.AddPipelineFlags(f, renderCmd, pipeline)
	return renderCmd
}
