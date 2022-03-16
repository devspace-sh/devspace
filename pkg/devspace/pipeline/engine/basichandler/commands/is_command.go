package commands

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/context/values"
	"mvdan.cc/sh/v3/interp"
	"os"
)

func IsCommand(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return interp.NewExitStatus(1)
	}

	command, ok := values.CommandFrom(ctx)
	if ok {
		if command == args[0] {
			return interp.NewExitStatus(0)
		}
		return interp.NewExitStatus(1)
	}

	if len(os.Args) < 2 {
		return interp.NewExitStatus(1)
	} else if os.Args[1] == args[0] {
		return interp.NewExitStatus(0)
	}
	return interp.NewExitStatus(1)
}
