package commands

import (
	"fmt"
	flags "github.com/jessevdk/go-flags"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/util/strvals"
	"github.com/pkg/errors"
	"io"
	"strings"
)

// DeployOptions describe how deployments should get deployed
type DeployOptions struct {
	deploy.Options

	Set       []string `long:"set" description:"Set configuration"`
	SetString []string `long:"set-string" description:"Set configuration as string"`
	From      []string `long:"from" description:"Reuse an existing configuration"`

	All bool `long:"all" description:"Deploy all deployments"`
}

func Deploy(ctx *devspacecontext.Context, pipeline types.Pipeline, args []string, stdout io.Writer) error {
	options := &DeployOptions{
		Options: pipeline.Options().DeployOptions,
	}
	args, err := flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}

	if options.All {
		deployments := ctx.Config.Config().Deployments
		for deployment := range deployments {
			ctx, err = applySetValues(ctx, "deployments", deployment, options.Set, options.SetString, options.From)
			if err != nil {
				return err
			}
		}
	} else if len(args) > 0 {
		for _, deployment := range args {
			ctx, err = applySetValues(ctx, "deployments", deployment, options.Set, options.SetString, options.From)
			if err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("either specify 'create_deployments --all' or 'create_deployments deployment1 deployment2'")
	}

	if options.RenderWriter == nil {
		options.RenderWriter = stdout
	}
	return deploy.NewController().Deploy(ctx, args, &options.Options)
}

func applySetValues(ctx *devspacecontext.Context, name, objName string, set, setString, from []string) (*devspacecontext.Context, error) {
	if len(set) == 0 && len(setString) == 0 && len(from) == 0 {
		return ctx, nil
	}

	rawConfigOriginal := ctx.Config.RawBeforeConversion()
	rawConfig := map[string]interface{}{}
	err := util.Convert(rawConfigOriginal, &rawConfig)
	if err != nil {
		return nil, err
	}

	_, ok := rawConfig[name].(map[string]interface{})
	if !ok {
		return ctx, nil
	}

	if rawConfig[name] == nil {
		rawConfig[name] = map[string]interface{}{}
	}
	if rawConfig[name].(map[string]interface{})[objName] == nil {
		rawConfig[name].(map[string]interface{})[objName] = map[string]interface{}{}
	}

	mapObj := rawConfig[name].(map[string]interface{})[objName].(map[string]interface{})
	for _, f := range from {
		if !matchesObjName(f, objName) {
			continue
		}
		if rawConfig[name].(map[string]interface{})[f] == nil {
			return nil, fmt.Errorf("couldn't find --from %s", f)
		}

		mapObj = strvals.MergeMaps(mapObj, rawConfig[name].(map[string]interface{})[f].(map[string]interface{}))
	}

	for _, s := range set {
		if !matchesObjName(s, objName) {
			continue
		}

		err = strvals.ParseInto(s, mapObj)
		if err != nil {
			return nil, errors.Wrap(err, "parsing --set flag")
		}
	}

	for _, s := range setString {
		if !matchesObjName(s, objName) {
			continue
		}

		err = strvals.ParseInto(s, mapObj)
		if err != nil {
			return nil, errors.Wrap(err, "parsing --set-string flag")
		}
	}

	latestConfig, err := loader.Convert(rawConfig, ctx.Log)
	if err != nil {
		return nil, err
	}

	return ctx.WithConfig(config.NewConfig(
		ctx.Config.Raw(),
		rawConfig,
		latestConfig,
		ctx.Config.LocalCache(),
		ctx.Config.RemoteCache(),
		ctx.Config.Variables(),
		ctx.Config.Path(),
	)), nil
}

func matchesObjName(s string, objName string) bool {
	splitted := strings.Split(s, ":")
	if len(splitted) > 1 && !strings.Contains(splitted[0], ".") && !strings.Contains(splitted[0], "=") && splitted[0] != objName {
		return false
	}
	return true
}
