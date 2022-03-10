package commands

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/devpod"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"strings"
)

// StopDevOptions describe how deployments should get deployed
type StopDevOptions struct {
	All bool `long:"all" description:"Stop all dev configurations"`
}

func StopDev(ctx *devspacecontext.Context, devManager devpod.Manager, args []string) error {
	ctx.Log.Debugf("stop_dev %s", strings.Join(args, " "))
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
