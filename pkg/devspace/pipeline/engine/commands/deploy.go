package commands

import (
	"fmt"
	flags "github.com/jessevdk/go-flags"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/pkg/errors"
)

// DeployOptions describe how deployments should get deployed
type DeployOptions struct {
	deploy.Options

	All bool `long:"all" description:"Deploy all deployments"`

	// Extra flags here to add an deployment
}

func Deploy(ctx *devspacecontext.Context, args []string) error {
	options := &DeployOptions{}
	args, err := flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}

	if !options.All && len(args) == 0 {
		return fmt.Errorf("deploy: either specify 'deploy --all' or 'deploy deployment1 deployment2'")
	}
	return deploy.NewController().Deploy(ctx, args, &options.Options)
}
