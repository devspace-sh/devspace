package commands

import (
	"mvdan.cc/sh/v3/interp"
	"strings"
)

func IsEmpty(args []string) error {
	// its possible that there are 0 args
	// because bash will omit an empty string
	// as argument
	if len(args) > 1 {
		return interp.NewExitStatus(1)
	} else if len(args) == 0 {
		return interp.NewExitStatus(0)
	}

	if strings.TrimSpace(args[0]) == "" {
		return interp.NewExitStatus(0)
	}
	return interp.NewExitStatus(1)
}
