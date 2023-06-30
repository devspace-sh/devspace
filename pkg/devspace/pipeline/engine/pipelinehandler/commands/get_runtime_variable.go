package commands

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/pkg/errors"
	"mvdan.cc/sh/v3/interp"
	"strings"
)

func GetRuntimeVariable(ctx devspacecontext.Context, args []string) error {
	ctx = ctx.WithLogger(ctx.Log().ErrorStreamOnly())
	ctx.Log().Debugf("get_image %s", strings.Join(args, " "))
	options := &GetImageOptions{}
	args, err := flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: get_runtime_variable [variable_name]")
	}

	_, runtimeVar, err := runtime.NewRuntimeVariable(args[0], ctx.Config(), ctx.Dependencies()).Load()
	if err != nil {
		return err
	}

	runtimeBytes, ok := runtimeVar.(string)
	if !ok {
		return fmt.Errorf("couldn't convert runtime variable %s to a string", args[0])
	}

	hc := interp.HandlerCtx(ctx.Context())
	_, _ = hc.Stdout.Write([]byte(runtimeBytes))
	return nil
}
