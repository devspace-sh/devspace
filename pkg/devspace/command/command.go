package command

import (
	"github.com/loft-sh/devspace/pkg/util/command"
	"github.com/loft-sh/devspace/pkg/util/shell"
	"os"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"

	"github.com/pkg/errors"
)

// ExecuteCommand executes a command from the config
func ExecuteCommand(commands []*latest.CommandConfig, name string, args []string) error {
	shellCommand := ""
	var shellArgs []string
	for _, command := range commands {
		if command.Name == name {
			shellCommand = command.Command
			shellArgs = command.Args
			break
		}
	}

	if shellCommand == "" {
		return errors.Errorf("couldn't find command '%s' in devspace config", name)
	}

	if shellArgs == nil {
		// Append args to shell command
		for _, arg := range args {
			arg = strings.Replace(arg, "'", "'\"'\"'", -1)

			shellCommand += " '" + arg + "'"
		}

		// execute the command in a shell
		return shell.ExecuteShellCommand(shellCommand, os.Stdout, os.Stderr, nil)
	}

	shellArgs = append(shellArgs, args...)
	return command.ExecuteCommand(shellCommand, shellArgs, os.Stdout, os.Stderr)
}
