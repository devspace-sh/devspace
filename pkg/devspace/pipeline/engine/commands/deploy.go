package commands

import (
	"fmt"
	flags "github.com/jessevdk/go-flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/pkg/errors"
)

// DeployOptions describe how deployments should get deployed
type DeployOptions struct {
	deploy.Options

	Set       []string `long:"set" description:"Set configuration"`
	SetString []string `long:"set-string" description:"Set configuration as string"`
	From      []string `long:"from" description:"Reuse an existing configuration"`

	All bool `long:"all" description:"Deploy all deployments"`

	// Extra flags here to add an deployment
}

func Deploy(ctx *devspacecontext.Context, args []string) error {
	options := &DeployOptions{}
	args, err := flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}

	ctx = ctx.WithConfig(ctx.Config.WithParsedConfig(ctx.Config.Config().Clone()))
	if options.All {
		for _, deployment := range ctx.Config.Config().Deployments {
			err = applyDeploymentSetValues(ctx.Config.Config(), deployment.Name, options.Set, options.SetString, options.From)
			if err != nil {
				return err
			}
		}
	} else if len(args) > 0 {
		for _, deployment := range args {
			err = applyDeploymentSetValues(ctx.Config.Config(), deployment, options.Set, options.SetString, options.From)
			if err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("deploy: either specify 'deploy --all' or 'deploy deployment1 deployment2'")
	}

	return deploy.NewController().Deploy(ctx, args, &options.Options)
}

func applyDeploymentSetValues(config *latest.Config, deployment string, set, setString, from []string) error {
	mapObj, err := applySetValues(deployment, set, setString, from, func(name string, create bool) (interface{}, error) {
		var (
			deploymentObj *latest.DeploymentConfig
		)
		for _, d := range config.Deployments {
			if d.Name == deployment {
				deploymentObj = d
				break
			}
		}
		if deploymentObj == nil {
			if !create {
				return nil, fmt.Errorf("couldn't find --from %s", name)
			}

			deploymentObj = &latest.DeploymentConfig{
				Name: deployment,
			}
			config.Deployments = append(config.Deployments, deploymentObj)
		}

		return deploymentObj, nil
	})
	if err != nil {
		return err
	}

	deploymentObj := &latest.DeploymentConfig{}
	err = util.Convert(mapObj, deploymentObj)
	if err != nil {
		return err
	}

	for i := range config.Deployments {
		if config.Deployments[i].Name == deployment {
			config.Deployments[i] = deploymentObj
			break
		}
	}
	return loader.Validate(config)
}
