package engine

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	enginecommands "github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/commands"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/commands/build"
	"github.com/loft-sh/devspace/pkg/util/downloader"
	"github.com/loft-sh/devspace/pkg/util/downloader/commands"
	"github.com/loft-sh/devspace/pkg/util/log"
	"mvdan.cc/sh/v3/interp"
	"os"
	"time"
)

type ExecHandler interface {
	ExecHandler(ctx context.Context, args []string) error
}

func NewExecHandler(devSpaceConfig config.Config, dependencies []types.Dependency, client kubectl.Client, logger log.Logger) ExecHandler {
	return &execHandler{
		devSpaceConfig: devSpaceConfig,
		dependencies:   dependencies,
		client:         client,
		logger:         logger,
	}
}

type execHandler struct {
	devSpaceConfig config.Config
	dependencies   []types.Dependency
	client         kubectl.Client
	logger         log.Logger
}

func (e *execHandler) ExecHandler(ctx context.Context, args []string) error {
	logger := log.GetFileLogger("shell")
	if len(args) > 0 {
		hc := interp.HandlerCtx(ctx)
		_, err := lookPathDir(hc.Dir, hc.Env, args[0])
		if err != nil {
			switch args[0] {
			case "cat":
				err = enginecommands.Cat(&hc, args[1:])
				if err != nil {
					_, _ = fmt.Fprintln(hc.Stderr, err)
					return interp.NewExitStatus(1)
				}
				return interp.NewExitStatus(0)
			case "kubectl":
				path, err := downloader.NewDownloader(commands.NewKubectlCommand(), logger).EnsureCommand()
				if err != nil {
					_, _ = fmt.Fprintln(hc.Stderr, err)
					return interp.NewExitStatus(127)
				}
				args[0] = path
			case "helm":
				path, err := downloader.NewDownloader(commands.NewHelmV3Command(), logger).EnsureCommand()
				if err != nil {
					_, _ = fmt.Fprintln(hc.Stderr, err)
					return interp.NewExitStatus(127)
				}
				args[0] = path
			case "devspace":
				if len(args) > 1 {
					switch args[1] {
					case "build":
						err = build.Build(e.devSpaceConfig, e.dependencies, e.client, log.NewStreamLogger(hc.Stdout, e.logger.GetLevel()), args[2:])
						if err != nil {
							_, _ = fmt.Fprintln(hc.Stderr, err)
							return interp.NewExitStatus(1)
						}
						return interp.NewExitStatus(0)
					}
				}

				bin, err := os.Executable()
				if err != nil {
					_, _ = fmt.Fprintln(hc.Stderr, err)
					return interp.NewExitStatus(1)
				}
				args[0] = bin
			default:
				_, _ = fmt.Fprintln(hc.Stderr, "command is not found.")
				return interp.NewExitStatus(127)
			}
		}
	}

	return interp.DefaultExecHandler(2*time.Second)(ctx, args)
}
