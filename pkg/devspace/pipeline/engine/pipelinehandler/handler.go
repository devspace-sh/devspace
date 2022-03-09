package pipelinehandler

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"

	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/basichandler"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/pipelinehandler/commands"
	enginetypes "github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/types"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"mvdan.cc/sh/v3/interp"
)

func NewPipelineExecHandler(ctx *devspacecontext.Context, stdout io.Writer, pipeline types.Pipeline, enablePipelineCommands bool) enginetypes.ExecHandler {
	return &execHandler{
		ctx:                    ctx,
		stdout:                 stdout,
		pipeline:               pipeline,
		enablePipelineCommands: enablePipelineCommands,

		basicHandler: basichandler.NewBasicExecHandler(),
	}
}

type execHandler struct {
	ctx                    *devspacecontext.Context
	stdout                 io.Writer
	pipeline               types.Pipeline
	enablePipelineCommands bool

	basicHandler enginetypes.ExecHandler
}

func (e *execHandler) ExecHandler(ctx context.Context, args []string) error {
	if len(args) > 0 {
		// handle special pipeline commands
		handled, err := e.handlePipelineCommands(ctx, args[0], args[1:])
		if handled || err != nil {
			return err
		}
	}

	return e.basicHandler.ExecHandler(ctx, args)
}

func (e *execHandler) handlePipelineCommands(ctx context.Context, command string, args []string) (bool, error) {
	hc := interp.HandlerCtx(ctx)
	devCtx := e.ctx.WithContext(ctx).
		WithWorkingDir(hc.Dir)
	if e.stdout != nil && e.stdout == hc.Stdout {
		devCtx = devCtx.WithLogger(e.ctx.Log)
	} else {
		devCtx = devCtx.WithLogger(log.NewStreamLogger(hc.Stdout, logrus.InfoLevel).WithoutPrefix())
	}

	switch command {
	case "run_command":
		return e.executePipelineCommand(ctx, command, func() error {
			return e.runCommand(devCtx, hc, args)
		})
	case "run_pipelines":
		return e.executePipelineCommand(ctx, command, func() error {
			return commands.Pipeline(devCtx, e.pipeline, args)
		})
	case "build_images":
		return e.executePipelineCommand(ctx, command, func() error {
			return commands.Build(devCtx, e.pipeline, args)
		})
	case "create_deployments":
		return e.executePipelineCommand(ctx, command, func() error {
			return commands.Deploy(devCtx, e.pipeline, args, hc.Stdout)
		})
	case "purge_deployments":
		return e.executePipelineCommand(ctx, command, func() error {
			return commands.Purge(devCtx, args)
		})
	case "start_dev":
		return e.executePipelineCommand(ctx, command, func() error {
			return commands.StartDev(devCtx, e.pipeline, args)
		})
	case "stop_dev":
		return e.executePipelineCommand(ctx, command, func() error {
			return commands.StopDev(devCtx, e.pipeline.DevPodManager(), args)
		})
	case "run_dependency_pipelines":
		return e.executePipelineCommand(ctx, command, func() error {
			return commands.Dependency(devCtx, e.pipeline, args)
		})
	case "ensure_pull_secrets":
		return e.executePipelineCommand(ctx, command, func() error {
			return commands.PullSecrets(devCtx, args)
		})
	}

	return false, nil
}

func (e *execHandler) executePipelineCommand(ctx context.Context, command string, commandFn func() error) (bool, error) {
	if e.pipeline == nil || !e.enablePipelineCommands {
		hc := interp.HandlerCtx(ctx)
		_, _ = fmt.Fprintln(hc.Stderr, fmt.Errorf("%s: cannot execute the command because it can only be executed within a pipeline step", command))
		return true, interp.NewExitStatus(1)
	}

	return true, handleError(ctx, command, commandFn())
}

func (e *execHandler) runCommand(ctx *devspacecontext.Context, hc interp.HandlerContext, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("please specify a command to run")
	}

	// try to find command
	for _, command := range ctx.Config.Config().Commands {
		if command.Name == args[0] {
			if len(command.Args) > 0 {
				return fmt.Errorf("calling commands that use args is not supported currently")
			}

			_, err := engine.ExecutePipelineShellCommand(ctx.Context, command.Command, args[1:], ctx.WorkingDir, false, hc.Stdout, hc.Stderr, hc.Stdin, hc.Env, NewPipelineExecHandler(ctx, hc.Stdout, e.pipeline, e.enablePipelineCommands))
			return err
		}
	}

	return fmt.Errorf("couldn't find command %v", args[0])
}

func handleError(ctx context.Context, command string, err error) error {
	if err == nil {
		return interp.NewExitStatus(0)
	}

	_, ok := interp.IsExitStatus(err)
	if ok {
		return err
	}

	hc := interp.HandlerCtx(ctx)
	_, _ = fmt.Fprintln(hc.Stderr, errors.Wrap(err, command))
	return interp.NewExitStatus(1)
}
