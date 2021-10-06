package command

import (
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/pkg/errors"

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
	RunWithEnv(stdout io.Writer, stderr io.Writer, stdin io.Reader, dir string, env map[string]string) error
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
func (f *FakeCommand) RunWithEnv(stdout io.Writer, stderr io.Writer, stdin io.Reader, dir string, extraEnvVars map[string]string) error {
	return nil
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
func (s *StreamCommand) RunWithEnv(stdout io.Writer, stderr io.Writer, stdin io.Reader, dir string, extraEnvVars map[string]string) error {
	s.cmd.Dir = dir
	env := os.Environ()
	for k, v := range extraEnvVars {
		env = append(env, k+"="+v)
	}

	s.cmd.Env = env
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

// Run runs a stream command
func (s *StreamCommand) Run(stdout io.Writer, stderr io.Writer, stdin io.Reader) error {
	return s.RunWithEnv(stdout, stderr, stdin, "", nil)
}

func ShouldExecuteOnOS(os string) bool {
	// if the operating system is set and the current is not specified
	// we skip the hook
	if os != "" {
		found := false
		oss := strings.Split(os, ",")
		for _, os := range oss {
			if strings.TrimSpace(os) == runtime.GOOS {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func ExecuteCommandWithEnv(cmd string, args []string, dir string, stdout io.Writer, stderr io.Writer, extraEnvVars map[string]string) error {
	err := NewStreamCommand(cmd, args).RunWithEnv(stdout, stderr, nil, dir, extraEnvVars)
	if err != nil {
		if errr, ok := err.(*exec.ExitError); ok {
			return errors.Errorf("error executing command '%s %s': code: %d, error: %s, %s", cmd, strings.Join(args, " "), errr.ExitCode(), string(errr.Stderr), errr)
		}

		return errors.Errorf("error executing command: %v", err)
	}

	return nil
}

func ExecuteCommand(cmd string, args []string, stdout io.Writer, stderr io.Writer) error {
	return ExecuteCommandWithEnv(cmd, args, "", stdout, stderr, nil)
}

func FormatCommandName(cmd string, args []string) string {
	commandString := strings.TrimSpace(cmd + " " + strings.Join(args, " "))
	splitted := strings.Split(commandString, "\n")
	if len(splitted) > 1 {
		return splitted[0] + "..."
	}

	return commandString
}
