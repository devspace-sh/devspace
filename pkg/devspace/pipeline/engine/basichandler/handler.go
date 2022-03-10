package basichandler

import (
	"context"
	"fmt"
	enginecommands "github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/basichandler/commands"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/types"
	"github.com/loft-sh/devspace/pkg/util/downloader"
	"github.com/loft-sh/devspace/pkg/util/downloader/commands"
	"github.com/loft-sh/devspace/pkg/util/log"
	"mvdan.cc/sh/v3/interp"
	"os"
	"time"
)

// BasicCommands are extra commands DevSpace provides within the shell or are common
// commands that might not be available locally for example in windows systems.
var BasicCommands = map[string]func(ctx context.Context, args []string) error{
	"is_os": func(ctx context.Context, args []string) error {
		return enginecommands.IsOS(args)
	},
	"is_equal": func(ctx context.Context, args []string) error {
		return enginecommands.IsEqual(args)
	},
	"is_command": func(ctx context.Context, args []string) error {
		return enginecommands.IsCommand(ctx, args)
	},
	"sleep": func(ctx context.Context, args []string) error {
		return handleError(interp.HandlerCtx(ctx), enginecommands.Sleep(args))
	},
	"cat": func(ctx context.Context, args []string) error {
		hc := interp.HandlerCtx(ctx)
		return handleError(hc, enginecommands.Cat(&hc, args))
	},
}

// EnsureCommands are commands where devspace makes sure those are installed locally before
// they can be used.
var EnsureCommands = map[string]func(ctx context.Context, args []string) (string, error){
	"kubectl": func(ctx context.Context, args []string) (string, error) {
		hc := interp.HandlerCtx(ctx)
		path, err := downloader.NewDownloader(commands.NewKubectlCommand(), log.GetFileLogger("shell")).EnsureCommand(ctx)
		if err != nil {
			_, _ = fmt.Fprintln(hc.Stderr, err)
			return "", interp.NewExitStatus(127)
		}
		return path, nil
	},
	"helm": func(ctx context.Context, args []string) (string, error) {
		hc := interp.HandlerCtx(ctx)
		path, err := downloader.NewDownloader(commands.NewHelmV3Command(), log.GetFileLogger("shell")).EnsureCommand(ctx)
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

func handleError(hc interp.HandlerContext, err error) error {
	if err != nil {
		_, _ = fmt.Fprintln(hc.Stderr, err)
		if _, ok := interp.IsExitStatus(err); ok {
			return err
		}
		return interp.NewExitStatus(1)
	}
	return interp.NewExitStatus(0)
}

var lookPathDir = interp.LookPathDir
