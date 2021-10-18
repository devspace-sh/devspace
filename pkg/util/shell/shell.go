package shell

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/loft-sh/devspace/pkg/util/downloader"
	"github.com/loft-sh/devspace/pkg/util/downloader/commands"
	"github.com/loft-sh/devspace/pkg/util/log"

	"github.com/pkg/errors"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

func ExecuteShellCommand(command string, args []string, dir string, stdout io.Writer, stderr io.Writer, extraEnvVars map[string]string) error {
	env := os.Environ()
	for k, v := range extraEnvVars {
		env = append(env, k+"="+v)
	}

	// Let's parse the complete command
	file, err := syntax.NewParser().Parse(strings.NewReader(command), "")
	if err != nil {
		return errors.Wrap(err, "parse shell command")
	}

	// Get current working directory
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	// Create shell runner
	r, err := interp.New(interp.Dir(dir), interp.StdIO(os.Stdin, stdout, stderr),
		interp.Env(expand.ListEnviron(env...)),
		interp.ExecHandler(DevSpaceExecHandler))
	if err != nil {
		return errors.Wrap(err, "create shell runner")
	}
	r.Params = args

	// Run command
	err = r.Run(context.Background(), file)
	if err != nil {
		if status, ok := interp.IsExitStatus(err); ok && status == 0 {
			return nil
		}

		return err
	}

	return nil
}

var lookPathDir = interp.LookPathDir

func DevSpaceExecHandler(ctx context.Context, args []string) error {
	logger := log.GetFileLogger("shell")
	if len(args) > 0 {
		hc := interp.HandlerCtx(ctx)
		_, err := lookPathDir(hc.Dir, hc.Env, args[0])
		if err != nil {
			switch args[0] {
			case "cat":
				err = cat(&hc, args[1:])
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
				bin, err := os.Executable()
				if err != nil {
					_, _ = fmt.Fprintln(hc.Stderr, err)
					return interp.NewExitStatus(1)
				}
				path := filepath.Dir(bin)
				args[0] = filepath.Join(path, os.Args[0])
			default:
				_, _ = fmt.Fprintln(hc.Stderr, "command is not found.")
				return interp.NewExitStatus(127)
			}
		}
	}

	return interp.DefaultExecHandler(2*time.Second)(ctx, args)
}
