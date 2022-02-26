package engine

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/env"
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

func ExecuteShellCommand(
	ctx context.Context,
	command string,
	args []string,
	dir string,
	continueOnError bool,
	stdout io.Writer,
	stderr io.Writer,
	environ expand.Environ,
	execHandler ExecHandler,
) (*interp.Runner, error) {
	// Replace runtime environment variables with ., so a runtime.images.test => runtime_images_test
	// which otherwise wouldn't be correct syntax
	command = replaceVariablesRegEx.ReplaceAllStringFunc(command, func(s string) string {
		if strings.Contains(s, ".") {
			return strings.ReplaceAll(s, ".", env.DotReplacement)
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
		interp.StdIO(os.Stdin, stdout, stderr),
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

var lookPathDir = interp.LookPathDir
