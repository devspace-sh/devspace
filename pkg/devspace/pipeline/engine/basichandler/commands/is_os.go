package commands

import (
	"mvdan.cc/sh/v3/interp"
	"runtime"
)

func IsOS(args []string) error {
	if len(args) > 1 {
		return interp.NewExitStatus(1)
	}

	// is empty string?
	if len(args) == 0 {
		return interp.NewExitStatus(1)
	}

	if args[0] == runtime.GOOS {
		return interp.NewExitStatus(0)
	}

	return interp.NewExitStatus(1)
}
