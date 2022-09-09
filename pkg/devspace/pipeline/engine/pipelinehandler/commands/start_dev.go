package commands

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/devpod"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/util/stringutil"
	"github.com/pkg/errors"
	"strings"
)

// StartDevOptions describe how deployments should get deployed
type StartDevOptions struct {
	devpod.Options

	Set       []string `long:"set" description:"Set configuration"`
	SetString []string `long:"set-string" description:"Set configuration as string"`
	From      []string `long:"from" description:"Reuse an existing configuration"`
	FromFile  []string `long:"from-file" description:"Reuse an existing configuration from a file"`

	All    bool     `long:"all" description:"Start all dev configurations"`
	Except []string `long:"except" description:"If used with --all, will exclude the following dev configs"`
}

func StartDev(ctx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
	ctx.Log().Debugf("start_dev %s", strings.Join(args, " "))
	err := pipeline.Exclude(ctx)
	if err != nil {
		return err
	}
	if ctx.KubeClient() == nil {
		return errors.Errorf(ErrMsg)
	}
	options := &StartDevOptions{
		Options: pipeline.Options().DevOptions,
	}
	args, err = flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}

	if options.All {
		args = []string{}
		for devConfig := range ctx.Config().Config().Dev {
			if stringutil.Contains(options.Except, devConfig) {
				continue
			}

			args = append(args, devConfig)
			ctx, err = applySetValues(ctx, "dev", devConfig, options.Set, options.SetString, options.From, options.FromFile)
			if err != nil {
				return err
			}
		}
		if len(args) == 0 {
			return nil
		}
	} else if len(args) > 0 {
		for _, devConfig := range args {
			ctx, err = applySetValues(ctx, "dev", devConfig, options.Set, options.SetString, options.From, options.FromFile)
			if err != nil {
				return err
			}

			if ctx.Config().Config().Dev == nil || ctx.Config().Config().Dev[devConfig] == nil {
				return fmt.Errorf("couldn't find dev %v", devConfig)
			}
		}
	} else {
		return fmt.Errorf("either specify 'start_dev --all' or 'dev devConfig1 devConfig2'")
	}
	return pipeline.DevPodManager().StartMultiple(ctx, args, options.Options)
}
