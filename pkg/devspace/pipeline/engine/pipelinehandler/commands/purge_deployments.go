package commands

import (
	"fmt"
	"strings"

	"github.com/jessevdk/go-flags"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/util/stringutil"
	"github.com/pkg/errors"
)

// PurgeDeploymentsOptions describe how deployments should get deployed
type PurgeDeploymentsOptions struct {
	deploy.PurgeOptions

	All    bool     `long:"all" description:"Deploy all deployments"`
	Except []string `long:"except" description:"If used with --all, will exclude the following deployments"`

	Sequential bool `long:"sequential" description:"Sequentially purges the deployments"`

	// Extra flags here to add a deployment
}

func PurgeDeployments(ctx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
	ctx.Log().Debugf("purge_deployments %s", strings.Join(args, " "))
	err := pipeline.Exclude(ctx)
	if err != nil {
		return err
	}
	if ctx.KubeClient() == nil {
		return errors.Errorf(ErrMsg)
	}
	options := &PurgeDeploymentsOptions{
		PurgeOptions: pipeline.Options().PurgeOptions,
	}
	args, err = flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}

	if !options.All && len(args) == 0 {
		return fmt.Errorf("either specify 'purge_deployments --all' or 'purge_deployments deployment1 deployment2'")
	} else if options.All {
		args = []string{}
		for _, d := range ctx.Config().RemoteCache().ListDeployments() {
			if stringutil.Contains(options.Except, d.Name) {
				continue
			}

			args = append(args, d.Name)
		}
		if len(args) == 0 {
			return nil
		}
	}

	return deploy.NewController().Purge(ctx, args, &options.PurgeOptions)
}
