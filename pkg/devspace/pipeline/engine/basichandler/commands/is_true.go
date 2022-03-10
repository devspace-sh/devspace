package commands

import (
	"mvdan.cc/sh/v3/interp"
)

func IsTrue(args []string) error {
	if len(args) != 1 {
		return interp.NewExitStatus(1)
	}

	if args[0] == "true" {
		return interp.NewExitStatus(0)
	}
	return interp.NewExitStatus(1)
}
