package pipelinehandler

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine"
	"io"
	"strings"

	"github.com/sirupsen/logrus"

	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/basichandler"
	basichandlercommands "github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/basichandler/commands"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/pipelinehandler/commands"
	enginetypes "github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/types"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/util/log"
	"mvdan.cc/sh/v3/interp"
)

// PipelineCommands are commands that can only be run within a pipeline and have special functionality in there
var PipelineCommands = map[string]func(devCtx devspacecontext.Context, pipeline types.Pipeline, args []string) error{
	"xargs": func(devCtx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
		hc := interp.HandlerCtx(devCtx.Context())
		return basichandlercommands.XArgs(devCtx.Context(), args, NewPipelineExecHandler(devCtx, hc.Stdout, hc.Stderr, pipeline))
	},
	"exec_container": func(devCtx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
		return commands.ExecContainer(devCtx, args)
	},
	"get_image": func(devCtx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
		return commands.GetImage(devCtx, args)
	},
	"run_default_pipeline": func(devCtx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
		return commands.RunDefaultPipeline(devCtx, pipeline, args, NewPipelineExecHandler)
	},
	"run_watch": func(devCtx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
		hc := interp.HandlerCtx(devCtx.Context())
		return basichandlercommands.RunWatch(devCtx.Context(), args, NewPipelineExecHandler(devCtx, hc.Stdout, hc.Stderr, pipeline), devCtx.Log())
	},
	"run_pipelines": func(devCtx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
		hc := interp.HandlerCtx(devCtx.Context())
		return commands.RunPipelines(devCtx, pipeline, args, hc.Env)
	},
	"build_images": func(devCtx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
		return commands.BuildImages(devCtx, pipeline, args)
	},
	"create_deployments": func(devCtx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
		hc := interp.HandlerCtx(devCtx.Context())
		return commands.CreateDeployments(devCtx, pipeline, args, hc.Stdout)
	},
	"purge_deployments": func(devCtx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
		return commands.PurgeDeployments(devCtx, pipeline, args)
	},
	"start_dev": func(devCtx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
		return commands.StartDev(devCtx, pipeline, args)
	},
	"stop_dev": func(devCtx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
		return commands.StopDev(devCtx, pipeline, args)
	},
	"run_dependencies": func(devCtx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
		return commands.RunDependencyPipelines(devCtx, pipeline, args)
	},
	"run_dependency_pipelines": func(devCtx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
		return commands.RunDependencyPipelines(devCtx, pipeline, args)
	},
	"ensure_pull_secrets": func(devCtx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
		return commands.EnsurePullSecrets(devCtx, pipeline, args)
	},
	"is_dependency": func(devCtx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
		return commands.IsDependency(devCtx.Context(), args)
	},
	"get_config_value": func(devCtx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
		return commands.GetConfigValue(devCtx, args)
	},
	"select_pod": func(devCtx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
		return commands.SelectPod(devCtx, args)
	},
	"wait_pod": func(devCtx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
		return commands.WaitPod(devCtx, args)
	},
}

func init() {
	// Add pipeline commands to basic handler to show an appropriate
	// error message if the command cannot be found due to running
	// outside of a pipeline
	for k := range PipelineCommands {
		name := k
		basichandler.BasicCommands[k] = func(ctx context.Context, args []string) error {
			hc := interp.HandlerCtx(ctx)
			_, _ = fmt.Fprintln(hc.Stderr, fmt.Errorf("cannot use command %s outside of a pipeline. Please make sure that you are calling %s within a pipeline execution. If you run a DevSpace command via `devspace run my-command` inside the pipeline, please use `run_command my-command` instead", name, name))
			return interp.NewExitStatus(1)
		}
	}
	for k := range PipelineCommands {
		basichandlercommands.XArgsFocusCommands[k] = true
	}
}

func NewPipelineExecHandler(ctx devspacecontext.Context, stdout, stderr io.Writer, pipeline types.Pipeline) enginetypes.ExecHandler {
	return &execHandler{
		ctx:      ctx,
		stdout:   stdout,
		stderr:   stderr,
		pipeline: pipeline,

		basicHandler: basichandler.NewBasicExecHandler(),
	}
}

type execHandler struct {
	ctx      devspacecontext.Context
	stdout   io.Writer
	stderr   io.Writer
	pipeline types.Pipeline

	basicHandler enginetypes.ExecHandler
}

func (e *execHandler) ExecHandler(ctx context.Context, args []string) error {
	select {
	case <-ctx.Done():
		return interp.NewExitStatus(255)
	default:
	}

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
	devCtx := e.ctx.WithContext(ctx).WithWorkingDir(hc.Dir).WithEnviron(hc.Env)
	if e.stdout != nil && e.stderr != nil && e.stdout == hc.Stdout && e.stderr == hc.Stderr {
		devCtx = devCtx.WithLogger(e.ctx.Log())
	} else {
		devCtx = devCtx.WithLogger(log.NewStreamLoggerWithFormat(hc.Stdout, hc.Stderr, logrus.InfoLevel, log.RawFormat))
	}

	// check if it's a function first
	if devCtx.Config().Config().Functions != nil {
		commandPayload, ok := devCtx.Config().Config().Functions[command]
		if ok {
			_, err := engine.ExecutePipelineShellCommand(devCtx.Context(), commandPayload, args, hc.Dir, false, hc.Stdout, hc.Stderr, hc.Stdin, hc.Env, NewPipelineExecHandler(devCtx, hc.Stdout, hc.Stderr, e.pipeline))
			return true, err
		}
	}

	// resolve pipeline commands
	pipelineCommand, ok := PipelineCommands[command]
	if ok {
		return e.executePipelineCommand(ctx, command, func() error {
			return pipelineCommand(devCtx, e.pipeline, args)
		})
	}

	// resolve internal pipeline commands
	pipelineCommand, ok = PipelineCommands[strings.TrimPrefix(command, "__")]
	if ok {
		return e.executePipelineCommand(ctx, command, func() error {
			return pipelineCommand(devCtx, e.pipeline, args)
		})
	}

	return false, nil
}

func (e *execHandler) executePipelineCommand(ctx context.Context, command string, commandFn func() error) (bool, error) {
	if e.pipeline == nil {
		hc := interp.HandlerCtx(ctx)
		_, _ = fmt.Fprintln(hc.Stderr, fmt.Errorf("%s: cannot execute the command because it can only be executed within a pipeline step", command))
		return true, interp.NewExitStatus(1)
	}

	return true, basichandler.HandleError(ctx, command, commandFn())
}
