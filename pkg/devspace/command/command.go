package command

import (
	"github.com/loft-sh/devspace/pkg/util/command"
	"github.com/loft-sh/devspace/pkg/util/shell"
	"io"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"

	"github.com/pkg/errors"
)

// ExecuteCommand executes a command from the config
func ExecuteCommand(commands []*latest.CommandConfig, name string, args []string, stdout io.Writer, stderr io.Writer) error {
	shellCommand := ""
	var shellArgs []string
	var appendArgs bool
	for _, cmd := range commands {
		if cmd.Name == name {
			shellCommand = cmd.Command
			shellArgs = cmd.Args
			appendArgs = cmd.AppendArgs
			break
		}
	}

	if shellCommand == "" {
		return errors.Errorf("couldn't find command '%s' in devspace config", name)
	}

	if shellArgs == nil {
		if appendArgs {
			// Append args to shell command
			for _, arg := range args {
				arg = strings.Replace(arg, "'", "'\"'\"'", -1)

				shellCommand += " '" + arg + "'"
			}
		}

		// execute the command in a shell
		return shell.ExecuteShellCommand(shellCommand, args, "", stdout, stderr, nil)
	}

	shellArgs = append(shellArgs, args...)
	return command.ExecuteCommand(shellCommand, shellArgs, stdout, stderr)
}
