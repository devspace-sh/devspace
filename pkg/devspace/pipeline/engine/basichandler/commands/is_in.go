package commands

import (
	"mvdan.cc/sh/v3/interp"
)

func IsIn(args []string) error {
	if len(args) < 2 {
		return interp.NewExitStatus(1)
	}
	needed := args[0]
	for _, value := range args[1:] {
		if value == needed {
			return interp.NewExitStatus(0)
		}
	}
	return interp.NewExitStatus(1)
}
