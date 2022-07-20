package commands

import (
	"fmt"
	flags "github.com/jessevdk/go-flags"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/devspace/pullsecrets"
	"github.com/loft-sh/devspace/pkg/util/stringutil"
	"github.com/pkg/errors"
	"strings"
)

// EnsurePullSecretsOptions describe how pull secrets should be deployed
type EnsurePullSecretsOptions struct {
	Set       []string `long:"set" description:"Set configuration"`
	SetString []string `long:"set-string" description:"Set configuration as string"`
	From      []string `long:"from" description:"Reuse an existing configuration"`
	FromFile  []string `long:"from-file" description:"Reuse an existing configuration from a file"`

	All    bool     `long:"all" description:"Ensure all pull secrets"`
	Except []string `long:"except" description:"If used with --all, will exclude the following pull secrets"`
}

func EnsurePullSecrets(ctx devspacecontext.Context, pipeline types.Pipeline, args []string) error {
	ctx.Log().Debugf("ensure_pull_secrets %s", strings.Join(args, " "))
	err := pipeline.Exclude(ctx)
	if err != nil {
		return err
	}
	if ctx.KubeClient() == nil {
		return errors.Errorf(ErrMsg)
	}
	options := &EnsurePullSecretsOptions{}
	args, err = flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}

	if options.All {
		if len(ctx.Config().Config().PullSecrets) == 0 {
			return nil
		}

		args = []string{}
		for pullSecret := range ctx.Config().Config().PullSecrets {
			if stringutil.Contains(options.Except, pullSecret) {
				continue
			}

			args = append(args, pullSecret)
			ctx, err = applySetValues(ctx, "pullSecrets", pullSecret, options.Set, options.SetString, options.From, options.FromFile)
			if err != nil {
				return err
			}
		}
		if len(args) == 0 {
			return nil
		}
	} else if len(args) > 0 {
		for _, pullSecret := range args {
			ctx, err = applySetValues(ctx, "pullSecrets", pullSecret, options.Set, options.SetString, options.From, options.FromFile)
			if err != nil {
				return err
			}

			if ctx.Config().Config().PullSecrets == nil || ctx.Config().Config().PullSecrets[pullSecret] == nil {
				return fmt.Errorf("couldn't find pull secret %v", pullSecret)
			}
		}
	} else {
		return fmt.Errorf("either specify 'ensure_pull_secrets --all' or 'ensure_pull_secrets pullSecret1 pullSecret2'")
	}

	dockerClient, err := docker.NewClient(ctx.Context(), ctx.Log())
	if err != nil {
		ctx.Log().Debugf("Error creating docker client: %v", err)
		dockerClient = nil
	}

	err = pullsecrets.NewClient().EnsurePullSecrets(ctx, dockerClient, args)
	if err != nil {
		return errors.Wrap(err, "ensure pull secrets")
	}

	return nil
}
