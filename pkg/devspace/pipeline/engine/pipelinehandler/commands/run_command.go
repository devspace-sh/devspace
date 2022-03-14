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

type NewHandlerFn func(ctx *devspacecontext.Context, stdout io.Writer, pipeline types.Pipeline) enginetypes.ExecHandler

func RunCommand(ctx *devspacecontext.Context, pipeline types.Pipeline, args []string, newHandler NewHandlerFn) error {
	hc := interp.HandlerCtx(ctx.Context)
	if len(args) == 0 {
		return fmt.Errorf("please specify a command to run")
	}

	// try to find command
	for _, command := range ctx.Config.Config().Commands {
		if command.Name == args[0] {
			if len(command.Args) > 0 {
				return fmt.Errorf("calling commands that use args is not supported currently")
			}

			_, err := engine.ExecutePipelineShellCommand(ctx.Context, command.Command, args[1:], hc.Dir, false, hc.Stdout, hc.Stderr, hc.Stdin, hc.Env, newHandler(ctx, hc.Stdout, pipeline))
			return err
		}
	}

	return fmt.Errorf("couldn't find command %v", args[0])
}
