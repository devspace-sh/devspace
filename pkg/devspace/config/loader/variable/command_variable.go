package variable

import (
	"bytes"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/command"
	"github.com/pkg/errors"
	"strings"
)

// NewCommandVariable creates a new variable that is loaded from a command
func NewCommandVariable(name string) Variable {
	return &commandVariable{
		name: name,
	}
}

type commandVariable struct {
	name string
}

func (c *commandVariable) Load(definition *latest.Variable) (interface{}, error) {
	if definition.Command == "" && len(definition.Commands) == 0 {
		return nil, errors.Errorf("couldn't set variable '%s', because source is '%s' but no command is specified", c.name, latest.VariableSourceCommand)
	}

	return variableFromCommand(c.name, definition)
}

func variableFromCommand(varName string, definition *latest.Variable) (interface{}, error) {
	for _, c := range definition.Commands {
		if command.ShouldExecuteOnOS(c.OperatingSystem) == false {
			continue
		}

		return execCommand(varName, definition, c.Command, c.Args)
	}
	if definition.Command == "" {
		return nil, errors.Errorf("couldn't set variable '%s', because source is '%s' but no command for this operating system is specified", varName, latest.VariableSourceCommand)
	}

	return execCommand(varName, definition, definition.Command, definition.Args)
}

func execCommand(varName string, definition *latest.Variable, cmd string, args []string) (interface{}, error) {
	writer := &bytes.Buffer{}
	stdErrWriter := &bytes.Buffer{}
	err := command.ExecuteCommand(cmd, args, writer, stdErrWriter)
	if err != nil {
		errMsg := "fill variable " + varName + ": " + err.Error()
		if len(writer.Bytes()) > 0 {
			errMsg = errMsg + "\n\nstdout: \n" + string(writer.Bytes())
		}
		if len(stdErrWriter.Bytes()) > 0 {
			errMsg = errMsg + "\n\nstderr: \n" + string(stdErrWriter.Bytes())
		}

		return "", errors.New(errMsg)
	} else if writer.String() == "" {
		return definition.Default, nil
	}

	return convertStringValue(strings.TrimSpace(writer.String())), nil
}
