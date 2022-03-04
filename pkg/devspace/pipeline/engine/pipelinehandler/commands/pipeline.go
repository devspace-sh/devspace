package commands

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/pkg/errors"
)

func Pipeline(ctx *devspacecontext.Context, pipeline types.Pipeline, args []string) error {
	options := &types.PipelineOptions{}
	args, err := flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}
	if len(args) == 0 {
		return fmt.Errorf("no pipeline to run specified")
	}

	pipelines := []*latest.Pipeline{}
	for _, arg := range args {
		pipelineConfig, ok := ctx.Config.Config().Pipelines[arg]
		if !ok {
			return fmt.Errorf("couldn't find pipeline %s", arg)
		}

		pipelines = append(pipelines, pipelineConfig)
	}

	return pipeline.StartNewPipelines(ctx, pipelines, *options)
}
