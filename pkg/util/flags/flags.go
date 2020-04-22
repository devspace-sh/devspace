package flags

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

// ApplyExtraFlags args parses the flags for a certain command from the environment variables
func ApplyExtraFlags(cobraCmd *cobra.Command) ([]string, error) {
	envName := strings.ToUpper(strings.Replace(cobraCmd.CommandPath(), " ", "_", -1) + "_FLAGS")

	flags, err := parseCommandLine(os.Getenv("DEVSPACE_FLAGS"))
	if err != nil {
		return nil, err
	}

	commandFlags, err := parseCommandLine(os.Getenv(envName))
	if err != nil {
		return nil, err
	}

	flags = append(flags, commandFlags...)
	if len(flags) == 0 {
		return nil, nil
	}

	err = cobraCmd.ParseFlags(flags)
	if err != nil {
		return nil, err
	}

	err = cobraCmd.ParseFlags(os.Args)
	if err != nil {
		return nil, err
	}

	return flags, nil
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
