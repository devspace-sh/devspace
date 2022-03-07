package commands

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/pkg/errors"
	"strings"
)

// PurgeOptions describe how deployments should get deployed
type PurgeOptions struct {
	All bool `long:"all" description:"Deploy all deployments"`

	Sequential bool `long:"sequential" description:"Sequentially purges the deployments"`

	// Extra flags here to add a deployment
}

func Purge(ctx *devspacecontext.Context, args []string) error {
	ctx.Log.Debugf("purge_deployments %s", strings.Join(args, " "))
	options := &PurgeOptions{}
	args, err := flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}

	if !options.All && len(args) == 0 {
		return fmt.Errorf("either specify 'purge_deployments --all' or 'purge_deployments deployment1 deployment2'")
	}

	return deploy.NewController().Purge(ctx, args)
}
