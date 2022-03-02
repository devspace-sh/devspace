package commands

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
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

	ctx = ctx.WithConfig(ctx.Config.WithParsedConfig(ctx.Config.Config().Clone()))
	if options.All {
		for devConfig := range ctx.Config.Config().Dev {
			err = applyDevSetValues(ctx.Config.Config(), devConfig, options.Set, options.SetString, options.From)
			if err != nil {
				return err
			}
		}
	} else if len(args) > 0 {
		for _, devConfig := range args {
			err = applyDevSetValues(ctx.Config.Config(), devConfig, options.Set, options.SetString, options.From)
			if err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("either specify 'start_dev --all' or 'dev devConfig1 devConfig2'")
	}
	return pipeline.DevPodManager().StartMultiple(ctx, args, options.Options)
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

	ctx = ctx.WithConfig(ctx.Config.WithParsedConfig(ctx.Config.Config().Clone()))
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

func applyDevSetValues(config *latest.Config, devConfig string, set, setString, from []string) error {
	if config.Dev == nil {
		config.Dev = map[string]*latest.DevPod{}
	}

	mapObj, err := applySetValues(devConfig, set, setString, from, func(name string, create bool) (interface{}, error) {
		imageObj, ok := config.Dev[devConfig]
		if !ok {
			if !create {
				return nil, fmt.Errorf("couldn't find --from %s", name)
			}

			return &latest.DevPod{
				Name: devConfig,
			}, nil
		}

		return imageObj, nil
	})
	if err != nil {
		return err
	}

	devConfigObj := &latest.DevPod{}
	err = util.Convert(mapObj, devConfigObj)
	if err != nil {
		return err
	}

	config.Dev[devConfig] = devConfigObj
	return loader.Validate(config)
}
