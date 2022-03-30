package cmd

import (
	"os"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewRenderCmd creates a new devspace render command
func NewRenderCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &RunPipelineCmd{
		GlobalFlags:  globalFlags,
		RenderWriter: os.Stdout,
	}

	renderCmd := &cobra.Command{
		Use:   "render",
		Short: "Render builds all defined images and shows the yamls that would be deployed",
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
			return cmd.Run(cobraCmd, args, f, "render", "renderCommand")
		},
	}

	cmd.AddFlags(renderCmd, "deploy")
	return renderCmd
}
