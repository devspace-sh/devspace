package variable

import (
	"bytes"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/command"
	"github.com/loft-sh/devspace/pkg/util/shell"
	"github.com/pkg/errors"
)

// NewCommandVariable creates a new variable that is loaded from a command
func NewCommandVariable(name, workingDirectory string) Variable {
	return &commandVariable{
		name:             name,
		workingDirectory: workingDirectory,
	}
}

type commandVariable struct {
	name             string
	workingDirectory string
}

func (c *commandVariable) Load(definition *latest.Variable) (interface{}, error) {
	if definition.Command == "" && len(definition.Commands) == 0 {
		return nil, errors.Errorf("couldn't set variable '%s', because source is '%s' but no command is specified", c.name, latest.VariableSourceCommand)
	}

	return variableFromCommand(c.name, c.workingDirectory, definition)
}

func variableFromCommand(varName string, dir string, definition *latest.Variable) (interface{}, error) {
	for _, c := range definition.Commands {
		if !command.ShouldExecuteOnOS(c.OperatingSystem) {
			continue
		}

		return execCommand(varName, definition, c.Command, c.Args, dir)
	}
	if definition.Command == "" {
		return nil, errors.Errorf("couldn't set variable '%s', because source is '%s' but no command for this operating system is specified", varName, latest.VariableSourceCommand)
	}

	return execCommand(varName, definition, definition.Command, definition.Args, dir)
}

func execCommand(varName string, definition *latest.Variable, cmd string, args []string, dir string) (interface{}, error) {
	writer := &bytes.Buffer{}
	stdErrWriter := &bytes.Buffer{}
	var err error
	if args == nil {
		err = shell.ExecuteShellCommand(cmd, nil, "", writer, stdErrWriter, nil)
	} else {
		err = command.ExecuteCommand(cmd, args, dir, writer, stdErrWriter)
	}
	if err != nil {
		errMsg := "fill variable " + varName + ": " + err.Error()
		if len(writer.Bytes()) > 0 {
			errMsg = errMsg + "\n\nstdout: \n" + writer.String()
		}
		if len(stdErrWriter.Bytes()) > 0 {
			errMsg = errMsg + "\n\nstderr: \n" + stdErrWriter.String()
		}

		return "", errors.New(errMsg)
	} else if writer.String() == "" {
		return definition.Default, nil
	}

	return convertStringValue(strings.TrimSpace(writer.String())), nil
}
