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

func NewBasicExecHandler() types.ExecHandler {
	return &execHandler{}
}

type execHandler struct{}

func (e *execHandler) ExecHandler(ctx context.Context, args []string) error {
	if len(args) > 0 {
		// handle some special commands that are not found locally
		hc := interp.HandlerCtx(ctx)
		_, err := lookPathDir(hc.Dir, hc.Env, args[0])
		if err != nil {
			err = e.fallbackCommands(ctx, args[0], args[1:])
			if err != nil {
				return err
			}
		}
	}

	return interp.DefaultExecHandler(2*time.Second)(ctx, args)
}

func (e *execHandler) executePipelineCommand(ctx context.Context, command string) (bool, error) {
	hc := interp.HandlerCtx(ctx)
	_, _ = fmt.Fprintln(hc.Stderr, fmt.Errorf("%s: cannot execute the command because it can only be executed within a pipeline step", command))
	return true, interp.NewExitStatus(1)
}

func (e *execHandler) fallbackCommands(ctx context.Context, command string, args []string) error {
	logger := log.GetFileLogger("shell")
	hc := interp.HandlerCtx(ctx)

	switch command {
	case "is_equal":
		return enginecommands.IsEqual(&hc, args)
	case "is_command":
		return enginecommands.IsCommand(args)
	case "sleep":
		return handleError(hc, enginecommands.Sleep(args))
	case "cat":
		return handleError(hc, enginecommands.Cat(&hc, args))
	case "kubectl":
		path, err := downloader.NewDownloader(commands.NewKubectlCommand(), logger).EnsureCommand(ctx)
		if err != nil {
			_, _ = fmt.Fprintln(hc.Stderr, err)
			return interp.NewExitStatus(127)
		}
		command = path
	case "helm":
		path, err := downloader.NewDownloader(commands.NewHelmV3Command(), logger).EnsureCommand(ctx)
		if err != nil {
			_, _ = fmt.Fprintln(hc.Stderr, err)
			return interp.NewExitStatus(127)
		}
		command = path
	case "devspace":
		bin, err := os.Executable()
		if err != nil {
			_, _ = fmt.Fprintln(hc.Stderr, err)
			return interp.NewExitStatus(1)
		}
		command = bin
	}
	return nil
}

func handleError(hc interp.HandlerContext, err error) error {
	if err != nil {
		_, _ = fmt.Fprintln(hc.Stderr, err)
		return interp.NewExitStatus(1)
	}
	return interp.NewExitStatus(0)
}

var lookPathDir = interp.LookPathDir
