package commands

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/devpod"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

// StartDevOptions describe how deployments should get deployed
type StartDevOptions struct {
	devpod.Options

	Set       []string `long:"set" description:"Set configuration"`
	SetString []string `long:"set-string" description:"Set configuration as string"`
	From      []string `long:"from" description:"Reuse an existing configuration"`

	All bool `long:"all" description:"Start all dev configurations"`
}

func StartDev(ctx *devspacecontext.Context, pipeline types.Pipeline, args []string) error {
	options := &StartDevOptions{
		Options: pipeline.Options().DevOptions,
	}
	args, err := flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}

	if options.All {
		for devConfig := range ctx.Config.Config().Dev {
			ctx, err = applySetValues(ctx, "dev", devConfig, options.Set, options.SetString, options.From)
			if err != nil {
				return err
			}
		}
	} else if len(args) > 0 {
		for _, devConfig := range args {
			ctx, err = applySetValues(ctx, "dev", devConfig, options.Set, options.SetString, options.From)
			if err != nil {
				return err
			}

			if ctx.Config.Config().Dev == nil || ctx.Config.Config().Dev[devConfig] == nil {
				return fmt.Errorf("couldn't find dev %v", devConfig)
			}
		}
	} else {
		return fmt.Errorf("either specify 'start_dev --all' or 'dev devConfig1 devConfig2'")
	}
	options.Options.KillApplication = func() {
		killApplication(pipeline)
	}
	return pipeline.DevPodManager().StartMultiple(ctx, args, options.Options)
}

func killApplication(pipeline types.Pipeline) {
	for pipeline.Parent() != nil {
		pipeline = pipeline.Parent()
	}

	err := pipeline.Close()
	if err != nil {
		logpkg.GetInstance().Errorf("error closing pipeline: %v", err)
	}
}

// StopDevOptions describe how deployments should get deployed
type StopDevOptions struct {
	All bool `long:"all" description:"Stop all dev configurations"`
}

func StopDev(ctx *devspacecontext.Context, devManager devpod.Manager, args []string) error {
	options := &StopDevOptions{}
	args, err := flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}

	if options.All {
		// loop over all pods in dev manager
		for _, a := range devManager.List() {
			ctx = ctx.WithLogger(logpkg.NewDefaultPrefixLogger("dev:"+a+" ", ctx.Log))
			ctx.Log.Infof("Stopping dev %s", a)
			err = devManager.Reset(ctx, a)
			if err != nil {
				return err
			}
		}

		// loop over all in cache
		for _, a := range ctx.Config.RemoteCache().ListDevPods() {
			ctx = ctx.WithLogger(logpkg.NewDefaultPrefixLogger("dev:"+a.Name+" ", ctx.Log))
			ctx.Log.Infof("Stopping dev %s", a.Name)
			err = devManager.Reset(ctx, a.Name)
			if err != nil {
				return err
			}
		}
	} else if len(args) > 0 {
		for _, a := range args {
			ctx = ctx.WithLogger(logpkg.NewDefaultPrefixLogger("dev:"+a+" ", ctx.Log))
			ctx.Log.Infof("Stopping dev %s", a)
			err = devManager.Reset(ctx, a)
			if err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("stop_dev: either specify 'stop_dev --all' or 'stop_dev devConfig1 devConfig2'")
	}

	return nil
}
