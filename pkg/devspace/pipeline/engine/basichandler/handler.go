package basichandler

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	enginecommands "github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/basichandler/commands"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/types"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/utils/pkg/downloader"
	"github.com/loft-sh/utils/pkg/downloader/commands"
	"github.com/pkg/errors"
	"mvdan.cc/sh/v3/interp"
)

func init() {
	for k := range BasicCommands {
		enginecommands.XArgsFocusCommands[k] = true
	}
	for k := range OverwriteCommands {
		enginecommands.XArgsFocusCommands[k] = true
	}
}

// BasicCommands are extra commands DevSpace provides within the shell or are common
// commands that might not be available locally for example in windows systems.
var BasicCommands = map[string]func(ctx context.Context, args []string) error{
	"get_flag": func(ctx context.Context, args []string) error {
		return enginecommands.GetFlag(ctx, args)
	},
	"is_os": func(ctx context.Context, args []string) error {
		return enginecommands.IsOS(args)
	},
	"is_equal": func(ctx context.Context, args []string) error {
		return enginecommands.IsEqual(args)
	},
	"is_empty": func(ctx context.Context, args []string) error {
		return enginecommands.IsEmpty(args)
	},
	"is_true": func(ctx context.Context, args []string) error {
		return enginecommands.IsTrue(args)
	},
	"is_in": func(ctx context.Context, args []string) error {
		return enginecommands.IsIn(args)
	},
	"sleep": func(ctx context.Context, args []string) error {
		return HandleError(ctx, "sleep", enginecommands.Sleep(ctx, args))
	},
	"cat": func(ctx context.Context, args []string) error {
		hc := interp.HandlerCtx(ctx)
		return HandleError(ctx, "cat", enginecommands.Cat(&hc, args))
	},
}

// OverwriteCommands are commands that overwrite existing bash commands
var OverwriteCommands = map[string]func(ctx context.Context, args []string, handler types.ExecHandler) error{
	"xargs": func(ctx context.Context, args []string, handler types.ExecHandler) error {
		return HandleError(ctx, "xargs", enginecommands.XArgs(ctx, args, handler))
	},
	"run_watch": func(ctx context.Context, args []string, handler types.ExecHandler) error {
		return HandleError(ctx, "run_watch", enginecommands.RunWatch(ctx, args, handler, log.Discard))
	},
}

// EnsureCommands are commands where devspace makes sure those are installed locally before
// they can be used.
var EnsureCommands = map[string]func(ctx context.Context, args []string) (string, error){
	"kubectl": func(ctx context.Context, args []string) (string, error) {
		hc := interp.HandlerCtx(ctx)
		path, err := downloader.NewDownloader(commands.NewKubectlCommand(), log.GetFileLogger("shell"), constants.DefaultHomeDevSpaceFolder).EnsureCommand(ctx)
		if err != nil {
			_, _ = fmt.Fprintln(hc.Stderr, err)
			return "", interp.NewExitStatus(127)
		}
		return path, nil
	},
	"helm": func(ctx context.Context, args []string) (string, error) {
		hc := interp.HandlerCtx(ctx)
		path, err := downloader.NewDownloader(commands.NewHelmV3Command(), log.GetFileLogger("shell"), constants.DefaultHomeDevSpaceFolder).EnsureCommand(ctx)
		if err != nil {
			_, _ = fmt.Fprintln(hc.Stderr, err)
			return "", interp.NewExitStatus(127)
		}
		return path, nil
	},
}

func NewBasicExecHandler() types.ExecHandler {
	return &execHandler{}
}

type execHandler struct{}

func (e *execHandler) ExecHandler(ctx context.Context, args []string) error {
	select {
	case <-ctx.Done():
		return interp.NewExitStatus(255)
	default:
	}

	if len(args) > 0 {
		hc := interp.HandlerCtx(ctx)

		// make sure if we reference devspace in a script we
		// always use the current binary
		if args[0] == "devspace" {
			bin, err := os.Executable()
			if err != nil {
				_, _ = fmt.Fprintln(hc.Stderr, err)
				return interp.NewExitStatus(1)
			}

			args[0] = bin
		} else {
			// handle overwrite commands
			overwriteCommand, ok := OverwriteCommands[args[0]]
			if ok {
				return overwriteCommand(ctx, args[1:], e)
			}

			// handle some special commands that are not found locally
			_, err := lookPathDir(hc.Dir, hc.Env, args[0])
			if err != nil {
				command, ok := BasicCommands[args[0]]
				if ok {
					return command(ctx, args[1:])
				}

				ensureCommand, ok := EnsureCommands[args[0]]
				if ok {
					path, err := ensureCommand(ctx, args[1:])
					if err != nil {
						return err
					} else if path != "" {
						args[0] = path
					}
				}
			}
		}
	}

	return interp.DefaultExecHandler(2*time.Second)(ctx, args)
}

func HandleError(ctx context.Context, command string, err error) error {
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

var lookPathDir = interp.LookPathDir
