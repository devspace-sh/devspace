package commands

import (
	"mvdan.cc/sh/v3/interp"
	"os"
)

func IsCommand(args []string) error {
	if len(args) != 1 || len(os.Args) < 2 {
		return interp.NewExitStatus(1)
	}

	if os.Args[1] == args[0] {
		return interp.NewExitStatus(0)
	}
	return interp.NewExitStatus(1)
}
