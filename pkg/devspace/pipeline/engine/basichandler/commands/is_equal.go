package commands

import (
	"mvdan.cc/sh/v3/interp"
	"strings"
)

func IsEqual(args []string) error {
	if len(args) > 2 {
		return interp.NewExitStatus(1)
	}

	// one of the arguments is an empty string
	if len(args) == 1 {
		if strings.TrimSpace(args[0]) == "" {
			return interp.NewExitStatus(0)
		}

		return interp.NewExitStatus(1)
	}

	// both arguments are empty strings
	if len(args) == 0 {
		return interp.NewExitStatus(0)
	}

	// compare arguments
	if args[0] == args[1] {
		return interp.NewExitStatus(0)
	}

	return interp.NewExitStatus(1)
}
