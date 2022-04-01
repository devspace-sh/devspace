package variable

import (
	"bytes"
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/command"
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

func (c *commandVariable) Load(ctx context.Context, definition *latest.Variable) (interface{}, error) {
	if definition.Command == "" && len(definition.Commands) == 0 {
		return nil, errors.Errorf("couldn't set variable '%s', because source is '%s' but no command is specified", c.name, latest.VariableSourceCommand)
	}

	return variableFromCommand(ctx, c.name, c.workingDirectory, definition)
}

func variableFromCommand(ctx context.Context, varName string, dir string, definition *latest.Variable) (interface{}, error) {
	for _, c := range definition.Commands {
		if !command.ShouldExecuteOnOS(c.OperatingSystem) {
			continue
		}

		return execCommand(ctx, varName, definition, c.Command, c.Args, dir)
	}
	if definition.Command == "" {
		return nil, errors.Errorf("couldn't set variable '%s', because source is '%s' but no command for this operating system is specified", varName, latest.VariableSourceCommand)
	}

	return execCommand(ctx, varName, definition, definition.Command, definition.Args, dir)
}

func execCommand(ctx context.Context, varName string, definition *latest.Variable, cmd string, args []string, dir string) (interface{}, error) {
	writer := &bytes.Buffer{}
	stdErrWriter := &bytes.Buffer{}
	var err error
	if args == nil {
		err = engine.ExecuteSimpleShellCommand(ctx, dir, writer, stdErrWriter, nil, nil, cmd)
	} else {
		err = command.Command(ctx, dir, writer, stdErrWriter, nil, cmd, args...)
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
