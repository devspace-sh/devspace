package kubectl

import (
	"os/exec"

	"github.com/loft-sh/devspace/pkg/util/command"
)

// These functions are detached from the DeployConfig so they can be faked by testers.

type commandExecuter interface {
	RunCommand(dir, path string, args []string) ([]byte, error)
	GetCommand(path string, args []string) command.Interface
}

type executer struct{}

func (e *executer) RunCommand(dir, path string, args []string) ([]byte, error) {
	cmd := exec.Command(path, args...)
	cmd.Dir = dir
	return cmd.Output()
}

func (e *executer) GetCommand(path string, args []string) command.Interface {
	return command.NewStreamCommand(path, args)
}
