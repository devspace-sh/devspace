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
	if len(args) == 0 {
		// handle some special commands that are not found locally
		hc := interp.HandlerCtx(ctx)
		_, err := lookPathDir(hc.Dir, hc.Env, args[0])
		if err != nil {
			logger := log.GetFileLogger("shell")
			hc := interp.HandlerCtx(ctx)
			switch args[0] {
			case "is_equal":
				return enginecommands.IsEqual(&hc, args[1:])
			case "is_command":
				return enginecommands.IsCommand(args[1:])
			case "sleep":
				return handleError(hc, enginecommands.Sleep(args[1:]))
			case "cat":
				return handleError(hc, enginecommands.Cat(&hc, args[1:]))
			case "kubectl":
				path, err := downloader.NewDownloader(commands.NewKubectlCommand(), logger).EnsureCommand(ctx)
				if err != nil {
					_, _ = fmt.Fprintln(hc.Stderr, err)
					return interp.NewExitStatus(127)
				}
				args[0] = path
			case "helm":
				path, err := downloader.NewDownloader(commands.NewHelmV3Command(), logger).EnsureCommand(ctx)
				if err != nil {
					_, _ = fmt.Fprintln(hc.Stderr, err)
					return interp.NewExitStatus(127)
				}
				args[0] = path
			case "devspace":
				bin, err := os.Executable()
				if err != nil {
					_, _ = fmt.Fprintln(hc.Stderr, err)
					return interp.NewExitStatus(1)
				}
				args[0] = bin
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
