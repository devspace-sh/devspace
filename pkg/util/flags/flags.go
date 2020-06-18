package flags

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

// ApplyExtraFlags args parses the flags for a certain command from the environment variables
func ApplyExtraFlags(cobraCmd *cobra.Command, osArgs []string, forceParsing bool) ([]string, error) {
	envName := strings.ToUpper(strings.Replace(cobraCmd.CommandPath(), " ", "_", -1) + "_FLAGS")

	flags, err := ParseCommandLine(os.Getenv("DEVSPACE_FLAGS"))
	if err != nil {
		return nil, err
	}

	commandFlags, err := ParseCommandLine(os.Getenv(envName))
	if err != nil {
		return nil, err
	}

	flags = append(flags, commandFlags...)
	if !forceParsing && len(flags) == 0 {
		return nil, nil
	}

	err = cobraCmd.ParseFlags(flags)
	if err != nil {
		return nil, err
	}

	err = cobraCmd.ParseFlags(osArgs)
	if err != nil {
		return nil, err
	}

	return flags, nil
}

// ParseCommandLine parses the command line string into an string array
func ParseCommandLine(command string) ([]string, error) {
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
