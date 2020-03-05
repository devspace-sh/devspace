package command

import (
	"io"
	"os/exec"

	goansi "github.com/k0kubun/go-ansi"
)

var defaultStdout = goansi.NewAnsiStdout()
var defaultStderr = goansi.NewAnsiStderr()

// Command is the default factory function
var Command Exec = NewStreamCommand

// Exec is the interface to create new commands
type Exec func(command string, args []string) Interface

// Interface is the command interface
type Interface interface {
	Run(stdout io.Writer, stderr io.Writer, stdin io.Reader) error
	Output() ([]byte, error)
	CombinedOutput() ([]byte, error)
}

// FakeCommand is used for testing
type FakeCommand struct {
	OutputBytes []byte
}

// CombinedOutput runs the command and returns the stdout and stderr
func (f *FakeCommand) CombinedOutput() ([]byte, error) {
	return f.OutputBytes, nil
}

// Output runs the command and returns the stdout
func (f *FakeCommand) Output() ([]byte, error) {
	return f.OutputBytes, nil
}

// Run implements interface
func (f *FakeCommand) Run(stdout io.Writer, stderr io.Writer, stdin io.Reader) error {
	return nil
}

// StreamCommand is the a command whose output is streamed to a log
type StreamCommand struct {
	cmd *exec.Cmd
}

// NewStreamCommand creates a new stram command
func NewStreamCommand(command string, args []string) Interface {
	return &StreamCommand{
		cmd: exec.Command(command, args...),
	}
}

// CombinedOutput runs the command and returns the stdout and stderr
func (s *StreamCommand) CombinedOutput() ([]byte, error) {
	return s.cmd.CombinedOutput()
}

// Output runs the command and returns the stdout
func (s *StreamCommand) Output() ([]byte, error) {
	return s.cmd.Output()
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
