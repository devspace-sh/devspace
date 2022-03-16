package commands

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/context/values"
	"mvdan.cc/sh/v3/interp"
)

func IsDependency(ctx context.Context, args []string) error {
	if len(args) > 0 {
		return interp.NewExitStatus(1)
	}

	isDependency, ok := values.IsDependencyFrom(ctx)
	if isDependency && ok {
		return interp.NewExitStatus(0)
	}
	return interp.NewExitStatus(1)
}
