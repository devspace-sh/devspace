package commands

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/context/values"
	"mvdan.cc/sh/v3/interp"
)

func GetFlag(ctx context.Context, args []string) error {
	hc := interp.HandlerCtx(ctx)
	if len(args) != 1 {
		_, _ = hc.Stderr.Write([]byte("usage: get_flag NAME"))
		return interp.NewExitStatus(1)
	}

	flags, ok := values.FlagsFrom(ctx)
	if !ok {
		_, _ = hc.Stderr.Write([]byte("cannot use get_flag in a non pipeline command"))
		return interp.NewExitStatus(1)
	}

	value, found := flags[args[0]]
	if !found {
		_, _ = hc.Stderr.Write([]byte(fmt.Sprintf("couldn't find flag %s", args[0])))
		return interp.NewExitStatus(1)
	}

	_, err := hc.Stdout.Write([]byte(value))
	if err != nil {
		_, _ = hc.Stderr.Write([]byte(err.Error()))
		return interp.NewExitStatus(1)
	}

	return interp.NewExitStatus(0)
}
