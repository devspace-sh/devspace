package command

import (
	"context"
	"os"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"

	"github.com/pkg/errors"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// ExecuteCommand executes a command from the config
func ExecuteCommand(config *latest.Config, name string, args []string) error {
	shellCommand := ""
	for _, command := range config.Commands {
		if command.Name == name {
			shellCommand = command.Command
			break
		}
	}

	if shellCommand == "" {
		return errors.Errorf("Couldn't find command '%s' in devspace.yaml", name)
	}

	// Append args to shell command
	for _, arg := range args {
		arg = strings.Replace(arg, "'", "'\"'\"'", -1)

		shellCommand += " '" + arg + "'"
	}

	// Let's parse the complete command
	file, err := syntax.NewParser().Parse(strings.NewReader(shellCommand), "")
	if err != nil {
		return errors.Wrap(err, "parse shell command")
	}

	// Get current working directory
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Create shell runner
	r, err := interp.New(interp.Dir(pwd), interp.StdIO(os.Stdin, os.Stdout, os.Stderr))
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
