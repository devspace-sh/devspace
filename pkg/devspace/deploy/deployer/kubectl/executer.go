package kubectl

import (
	"os/exec"

	"github.com/loft-sh/devspace/pkg/util/command"
)

// These functions are detached from the DeployConfig so they can be faked by testers.

type commandExecuter interface {
	RunCommand(path string, args []string) ([]byte, error)
	GetCommand(path string, args []string) command.Interface
}

type executer struct{}

func (e *executer) RunCommand(path string, args []string) ([]byte, error) {
	return exec.Command(path, args...).Output()
}

func (e *executer) GetCommand(path string, args []string) command.Interface {
	return command.NewStreamCommand(path, args)
}
