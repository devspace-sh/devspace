package commands

import (
	"fmt"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"mvdan.cc/sh/v3/interp"
)

func RunDefaultPipeline(ctx *devspacecontext.Context, pipeline types.Pipeline, args []string, newHandler NewHandlerFn) error {
	hc := interp.HandlerCtx(ctx.Context)
	if len(args) != 1 {
		return fmt.Errorf("usage: run_default_pipeline [pipeline]")
	}

	defaultPipeline, err := types.GetDefaultPipeline(args[0])
	if err != nil {
		return err
	}

	_, err = engine.ExecutePipelineShellCommand(ctx.Context, defaultPipeline.Steps[0].Run, nil, hc.Dir, false, hc.Stdout, hc.Stderr, hc.Stdin, hc.Env, newHandler(ctx, hc.Stdout, pipeline))
	return err
}
