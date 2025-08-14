package cmd

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewDeployCmd creates a new deploy command
func NewDeployCmd(f factory.Factory, globalFlags *flags.GlobalFlags, rawConfig *RawConfig) *cobra.Command {
	cmd := &RunPipelineCmd{
		GlobalFlags:             globalFlags,
		SkipPushLocalKubernetes: true,
		Pipeline:                "deploy",
	}

	var pipeline *latest.Pipeline
	if rawConfig != nil && rawConfig.Config != nil && rawConfig.Config.Pipelines != nil {
		pipeline = rawConfig.Config.Pipelines["deploy"]
	}
	deployCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploys the project",
		Long: `
#######################################################
################## devspace deploy ####################
#######################################################
Deploys the current project to a Space or namespace:

devspace deploy
devspace deploy -n some-namespace
devspace deploy --kube-context=deploy-context
#######################################################`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd, args, f, "deployCommand")
		},
	}
	cmd.AddPipelineFlags(f, deployCmd, pipeline)
	return deployCmd
}
