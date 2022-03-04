package commands

import (
	"fmt"
	flags "github.com/jessevdk/go-flags"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/pullsecrets"
	"github.com/pkg/errors"
)

// PullSecretsOptions describe how pull secrets should be deployed
type PullSecretsOptions struct {
	Set       []string `long:"set" description:"Set configuration"`
	SetString []string `long:"set-string" description:"Set configuration as string"`
	From      []string `long:"from" description:"Reuse an existing configuration"`

	All bool `long:"all" description:"Ensure all pull secrets"`
}

func PullSecrets(ctx *devspacecontext.Context, args []string) error {
	options := &PullSecretsOptions{}
	args, err := flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}

	if options.All {
		pullSecrets := ctx.Config.Config().PullSecrets
		if len(pullSecrets) == 0 {
			return nil
		}

		for pullSecret := range pullSecrets {
			ctx, err = applySetValues(ctx, "pullSecrets", pullSecret, options.Set, options.SetString, options.From)
			if err != nil {
				return err
			}
		}
	} else if len(args) > 0 {
		for _, pullSecret := range args {
			ctx, err = applySetValues(ctx, "pullSecrets", pullSecret, options.Set, options.SetString, options.From)
			if err != nil {
				return err
			}

			if ctx.Config.Config().PullSecrets == nil || ctx.Config.Config().PullSecrets[pullSecret] == nil {
				return fmt.Errorf("couldn't find pull secret %v", pullSecret)
			}
		}
	} else {
		return fmt.Errorf("either specify 'ensure_pull_secrets --all' or 'ensure_pull_secrets pullSecret1 pullSecret2'")
	}

	dockerClient, err := docker.NewClient(ctx.Log)
	if err != nil {
		ctx.Log.Debugf("Error creating docker client: %v", err)
		dockerClient = nil
	}

	err = pullsecrets.NewClient().EnsurePullSecrets(ctx, dockerClient, args)
	if err != nil {
		return errors.Wrap(err, "ensure pull secrets")
	}

	return nil
}
