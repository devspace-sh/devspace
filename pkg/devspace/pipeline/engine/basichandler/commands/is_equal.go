package commands

import (
	"mvdan.cc/sh/v3/interp"
)

func IsEqual(args []string) error {
	if len(args) != 2 {
		return interp.NewExitStatus(1)
	}

	if args[0] == args[1] {
		return interp.NewExitStatus(0)
	}

	return interp.NewExitStatus(1)
}
