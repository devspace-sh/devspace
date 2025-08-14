package commands

import (
	"strings"

	"mvdan.cc/sh/v3/interp"
)

func IsIn(args []string) error {
	if len(args) != 2 {
		return interp.NewExitStatus(1)
	}
	values := strings.Split(args[1], " ")
	for _, value := range values {
		if value == args[0] {
			return interp.NewExitStatus(0)
		}
	}
	return interp.NewExitStatus(1)
}
