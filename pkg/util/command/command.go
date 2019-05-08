package command

import (
	"io"
	"os/exec"

	goansi "github.com/k0kubun/go-ansi"
)

var defaultStdout = goansi.NewAnsiStdout()
var defaultStderr = goansi.NewAnsiStderr()

// Interface is the command interface
type Interface interface {
	Run(stdout io.Writer, stderr io.Writer, stdin io.Reader) error
}

// FakeCommand is used for testing
type FakeCommand struct{}

// Run implements interface
func (f *FakeCommand) Run(stdout io.Writer, stderr io.Writer, stdin io.Reader) error {
	return nil
}

// StreamCommand is the a command whose output is streamed to a log
type StreamCommand struct {
	cmd *exec.Cmd
}

// NewStreamCommand creates a new stram command
func NewStreamCommand(command string, args []string) *StreamCommand {
	cmd := exec.Command(command, args...)

	return &StreamCommand{
		cmd: cmd,
	}
}

// Run runs a stream command
func (s *StreamCommand) Run(stdout io.Writer, stderr io.Writer, stdin io.Reader) error {
	if stdout == nil {
		s.cmd.Stdout = defaultStdout
	} else {
		s.cmd.Stdout = stdout
	}

	if stderr == nil {
		s.cmd.Stderr = defaultStderr
	} else {
		s.cmd.Stderr = stderr
	}

	if stdin != nil {
		s.cmd.Stdin = stdin
	}

	return s.cmd.Run()
}
