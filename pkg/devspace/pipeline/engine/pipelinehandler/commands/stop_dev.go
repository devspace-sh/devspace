package commands

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/util/stringutil"
	"github.com/pkg/errors"
	"strings"
)

// StopDevOptions describe how deployments should get deployed
type StopDevOptions struct {
	deploy.PurgeOptions

	All    bool     `long:"all" description:"Stop all dev configurations"`
	Except []string `long:"except" description:"If used with --all, will exclude the following dev configs"`
}

func StopDev(ctx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
	ctx.Log().Debugf("stop_dev %s", strings.Join(args, " "))
	err := pipeline.Exclude(ctx)
	if err != nil {
		return err
	}
	if ctx.KubeClient() == nil {
		return errors.Errorf(ErrMsg)
	}
	options := &StopDevOptions{
		PurgeOptions: pipeline.Options().PurgeOptions,
	}
	args, err = flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}

	devManager := pipeline.DevPodManager()
	if options.All {
		// loop over all pods in dev manager
		for _, a := range devManager.List() {
			if stringutil.Contains(options.Except, a) {
				continue
			}

			ctx = ctx.WithLogger(ctx.Log().WithPrefix("dev:" + a + " "))
			ctx.Log().Infof("Stopping dev %s", a)
			err = devManager.Reset(ctx, a, &options.PurgeOptions)
			if err != nil {
				return err
			}
		}

		// loop over all in cache
		for _, a := range ctx.Config().RemoteCache().ListDevPods() {
			if stringutil.Contains(options.Except, a.Name) {
				continue
			}

			ctx = ctx.WithLogger(ctx.Log().WithPrefix("dev:" + a.Name + " "))
			ctx.Log().Infof("Stopping dev %s", a.Name)
			err = devManager.Reset(ctx, a.Name, &options.PurgeOptions)
			if err != nil {
				return err
			}
		}
	} else if len(args) > 0 {
		for _, a := range args {
			ctx = ctx.WithLogger(ctx.Log().WithPrefix("dev:" + a + " "))
			ctx.Log().Infof("Stopping dev %s", a)
			err = devManager.Reset(ctx, a, &options.PurgeOptions)
			if err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("stop_dev: either specify 'stop_dev --all' or 'stop_dev devConfig1 devConfig2'")
	}

	return nil
}
