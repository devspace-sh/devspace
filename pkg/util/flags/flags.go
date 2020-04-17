package flags

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type Flags interface {
	Apply() []string
	Parse(command string) error
}

// flags is a helper struct to parse flags from the environment
type flags struct {
	args []string
}

// New creates a new flag set
func New() Flags {
	return &flags{args: []string{}}
}

// Apply appends the flags to the os.Args
func (f *flags) Apply() []string {
	if len(f.args) == 0 {
		return nil
	}

	newArgs := []string{os.Args[0]}
	newArgs = append(newArgs, f.args...)
	newArgs = append(newArgs, os.Args[1:]...)
	os.Args = newArgs
	return f.args
}

// Parse args parses the flags for a certain command from the environment variables
func (f *flags) Parse(command string) error {
	if command != "" && isCommand(command) == false {
		return nil
	}

	envName := strings.ToUpper("DEVSPACE_" + command + "_FLAGS")
	if command == "" {
		envName = "DEVSPACE_FLAGS"
	}

	flags, err := parseCommandLine(os.Getenv(envName))
	if err != nil {
		return err
	}

	f.args = append(f.args, flags...)
	return nil
}

func isCommand(command string) bool {
	return len(os.Args) > 1 && os.Args[1] == command
}

func parseCommandLine(command string) ([]string, error) {
	var args []string
	state := "start"
	current := ""
	quote := "\""
	escapeNext := true
	for i := 0; i < len(command); i++ {
		c := command[i]

		if state == "quotes" {
			if string(c) != quote {
				current += string(c)
			} else {
				args = append(args, current)
				current = ""
				state = "start"
			}
			continue
		}

		if escapeNext {
			current += string(c)
			escapeNext = false
			continue
		}

		if c == '\\' {
			escapeNext = true
			continue
		}

		if c == '"' || c == '\'' {
			state = "quotes"
			quote = string(c)
			continue
		}

		if state == "arg" {
			if c == ' ' || c == '\t' {
				args = append(args, current)
				current = ""
				state = "start"
			} else {
				current += string(c)
			}
			continue
		}

		if c != ' ' && c != '\t' {
			state = "arg"
			current += string(c)
		}
	}

	if state == "quotes" {
		return []string{}, errors.New(fmt.Sprintf("Unclosed quote in command line: %s", command))
	}

	if current != "" {
		args = append(args, current)
	}

	return args, nil
}
