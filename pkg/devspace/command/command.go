package command

import (
	"context"
	"io"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
	"os"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"

	"github.com/pkg/errors"
)

// ExecuteCommand executes a command from the config
func ExecuteCommand(commands []*latest.CommandConfig, name string, args []string) error {
	shellCommand := ""
	for _, command := range commands {
		if command.Name == name {
			shellCommand = command.Command
			break
		}
	}

	if shellCommand == "" {
		return errors.Errorf("couldn't find command '%s' in devspace config", name)
	}

	// Append args to shell command
	for _, arg := range args {
		arg = strings.Replace(arg, "'", "'\"'\"'", -1)

		shellCommand += " '" + arg + "'"
	}

	// execute the command
	return ExecuteShellCommand(shellCommand, os.Stdout, os.Stderr, nil)
}

func ExecuteShellCommand(command string, stdout io.Writer, stderr io.Writer, extraEnvVars map[string]string) error {
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
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Create shell runner
	r, err := interp.New(interp.Dir(pwd), interp.StdIO(os.Stdin, stdout, stderr), interp.Env(expand.ListEnviron(env...)))
	if err != nil {
		return errors.Wrap(err, "create shell runner")
	}

	// Run command
	err = r.Run(context.Background(), file)
	if err != nil && err != interp.ShellExitStatus(0) {
		return err
	}

	return nil
}
