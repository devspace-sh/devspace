package commands

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/devpod"
	"github.com/pkg/errors"
)

// DevOptions describe how deployments should get deployed
type DevOptions struct {
	All bool `long:"all" description:"Start all dev configurations"`
}

func Dev(ctx *devspacecontext.Context, devManager devpod.Manager, args []string) error {
	options := &DevOptions{}
	args, err := flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}

	if !options.All && len(args) == 0 {
		return fmt.Errorf("dev: either specify 'dev --all' or 'dev devConfig1 devConfig2'")
	}
	return devManager.StartMultiple(ctx, args)
}
