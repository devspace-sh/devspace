package engine

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/basichandler"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/types"
	"github.com/pkg/errors"
	"io"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
	"os"
	"regexp"
	"strings"
)

var replaceVariablesRegEx = regexp.MustCompile(`\$\{[a-zA-Z_.]+?\}`)

func ExecuteSimpleShellCommand(
	ctx context.Context,
	dir string,
	environ expand.Environ,
	stdout io.Writer,
	stderr io.Writer,
	stdin io.Reader,
	command string,
	args ...string,
) error {
	_, err := ExecutePipelineShellCommand(ctx, command, args, dir, false, stdout, stderr, stdin, environ, basichandler.NewBasicExecHandler())
	return err
}

func ExecutePipelineShellCommand(
	ctx context.Context,
	command string,
	args []string,
	dir string,
	continueOnError bool,
	stdout io.Writer,
	stderr io.Writer,
	stdin io.Reader,
	environ expand.Environ,
	execHandler types.ExecHandler,
) (*interp.Runner, error) {
	// Replace runtime environment variables with ., so a runtime.images.test => runtime_images_test
	// which otherwise wouldn't be correct syntax
	command = replaceVariablesRegEx.ReplaceAllStringFunc(command, func(s string) string {
		if strings.Contains(s, ".") {
			return strings.ReplaceAll(s, ".", types.DotReplacement)
		}

		return s
	})

	// Let's parse the complete command
	file, err := syntax.NewParser().Parse(strings.NewReader(command), "")
	if err != nil {
		return nil, errors.Wrap(err, "parse shell command")
	}

	// Get current working directory
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	// create options
	options := []interp.RunnerOption{
		interp.Dir(dir),
		interp.StdIO(stdin, stdout, stderr),
		interp.Env(environ),
		interp.ExecHandler(execHandler.ExecHandler),
	}
	if !continueOnError {
		options = append(options, interp.Params("-e"))
	}

	// Create shell runner
	r, err := interp.New(options...)
	if err != nil {
		return nil, errors.Wrap(err, "create shell runner")
	}
	r.Params = args

	// Run command
	err = r.Run(ctx, file)
	if err != nil {
		if status, ok := interp.IsExitStatus(err); ok && status == 0 {
			return r, nil
		}

		return r, err
	}

	return r, nil
}
