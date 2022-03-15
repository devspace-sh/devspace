package commands

import (
	"fmt"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine"
	enginetypes "github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/types"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"io"
	"mvdan.cc/sh/v3/interp"
)

type NewHandlerFn func(ctx *devspacecontext.Context, stdout, stderr io.Writer, pipeline types.Pipeline) enginetypes.ExecHandler

func RunDefaultPipeline(ctx *devspacecontext.Context, pipeline types.Pipeline, args []string, newHandler NewHandlerFn) error {
	hc := interp.HandlerCtx(ctx.Context)
	if len(args) != 1 {
		return fmt.Errorf("usage: run_default_pipeline [pipeline]")
	}

	defaultPipeline, err := types.GetDefaultPipeline(args[0])
	if err != nil {
		return err
	}

	_, err = engine.ExecutePipelineShellCommand(ctx.Context, defaultPipeline.Run, nil, hc.Dir, false, hc.Stdout, hc.Stderr, hc.Stdin, hc.Env, newHandler(ctx, hc.Stdout, hc.Stderr, pipeline))
	return err
}
