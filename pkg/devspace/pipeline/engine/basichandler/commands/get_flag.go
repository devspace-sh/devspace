package commands

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/context/values"
	flag "github.com/spf13/pflag"
	"mvdan.cc/sh/v3/interp"
	"strings"
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

	value := ""
	found := false
	flags.VisitAll(func(f *flag.Flag) {
		if !found && f.Name == args[0] {
			sliceType, ok := f.Value.(flag.SliceValue)
			if ok {
				value = strings.Join(sliceType.GetSlice(), " ")
			} else {
				value = f.Value.String()
			}
			found = true
		}
	})
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
