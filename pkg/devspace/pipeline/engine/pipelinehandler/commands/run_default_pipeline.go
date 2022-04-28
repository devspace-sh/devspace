package commands

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine"
	enginetypes "github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/types"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/util/strvals"
	"github.com/pkg/errors"
	"io"
	"mvdan.cc/sh/v3/interp"
)

type RunDefaultPipelineOptions struct {
	Set       []string `long:"set" description:"Set configuration"`
	SetString []string `long:"set-string" description:"Set configuration as string"`
}

type NewHandlerFn func(ctx devspacecontext.Context, stdout, stderr io.Writer, pipeline types.Pipeline) enginetypes.ExecHandler

func RunDefaultPipeline(ctx devspacecontext.Context, pipeline types.Pipeline, args []string, newHandler NewHandlerFn) error {
	err := pipeline.Exclude(ctx)
	if err != nil {
		return err
	}

	hc := interp.HandlerCtx(ctx.Context())

	options := &RunDefaultPipelineOptions{}
	args, err = flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}

	if len(args) != 1 {
		return fmt.Errorf("usage: run_default_pipeline [pipeline]")
	}

	if len(args) > 0 {
		ctx, err = applyPipelineSetValue(ctx, options.Set, options.SetString)
		if err != nil {
			return err
		}
	}

	defaultPipeline, err := types.GetDefaultPipeline(args[0])
	if err != nil {
		return err
	}

	_, err = engine.ExecutePipelineShellCommand(ctx.Context(), defaultPipeline.Run, nil, hc.Dir, false, hc.Stdout, hc.Stderr, hc.Stdin, hc.Env, newHandler(ctx, hc.Stdout, hc.Stderr, pipeline))
	return err
}

func applyPipelineSetValue(ctx devspacecontext.Context, set, setString []string) (devspacecontext.Context, error) {
	if len(set) == 0 && len(setString) == 0 {
		return ctx, nil
	}

	rawConfigOriginal := ctx.Config().RawBeforeConversion()
	rawConfig := map[string]interface{}{}
	err := util.Convert(rawConfigOriginal, &rawConfig)
	if err != nil {
		return nil, err
	}

	for _, s := range set {
		err = strvals.ParseInto(s, rawConfig)
		if err != nil {
			return nil, errors.Wrap(err, "parsing --set flag")
		}
	}

	for _, s := range setString {
		err = strvals.ParseIntoString(s, rawConfig)
		if err != nil {
			return nil, errors.Wrap(err, "parsing --set-string flag")
		}
	}

	latestConfig, err := versions.Parse(rawConfig, ctx.Log())
	if err != nil {
		return nil, err
	}

	return ctx.WithConfig(config.NewConfig(
		ctx.Config().Raw(),
		rawConfig,
		latestConfig,
		ctx.Config().LocalCache(),
		ctx.Config().RemoteCache(),
		ctx.Config().Variables(),
		ctx.Config().Path(),
	)), nil
}
