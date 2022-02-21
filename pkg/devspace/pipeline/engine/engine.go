package engine

import (
	"context"
	"github.com/pkg/errors"
	"io"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
	"os"
	"strings"
)

func ExecuteShellCommand(command string, args []string, dir string, stdout io.Writer, stderr io.Writer, extraEnvVars map[string]string, execHandler ExecHandler) (*interp.Runner, error) {
	env := os.Environ()
	for k, v := range extraEnvVars {
		env = append(env, k+"="+v)
	}

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

	// Create shell runner
	r, err := interp.New(interp.Dir(dir), interp.StdIO(os.Stdin, stdout, stderr),
		interp.Env(expand.ListEnviron(env...)),
		interp.ExecHandler(execHandler.ExecHandler))
	if err != nil {
		return nil, errors.Wrap(err, "create shell runner")
	}
	r.Params = args

	// Run command
	err = r.Run(context.Background(), file)
	if err != nil {
		if status, ok := interp.IsExitStatus(err); ok && status == 0 {
			return r, nil
		}

		return r, err
	}

	return r, nil
}

var lookPathDir = interp.LookPathDir
