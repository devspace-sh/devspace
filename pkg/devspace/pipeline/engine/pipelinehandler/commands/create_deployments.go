package commands

import (
	"fmt"
	flags "github.com/jessevdk/go-flags"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/util/stringutil"
	"github.com/loft-sh/devspace/pkg/util/strvals"
	"github.com/loft-sh/devspace/pkg/util/yamlutil"
	"github.com/pkg/errors"
	"io"
	"os"
	"strings"
)

// CreateDeploymentsOptions describe how deployments should get deployed
type CreateDeploymentsOptions struct {
	deploy.Options

	Set       []string `long:"set" description:"Set configuration"`
	SetString []string `long:"set-string" description:"Set configuration as string"`
	From      []string `long:"from" description:"Reuse an existing configuration"`
	FromFile  []string `long:"from-file" description:"Reuse an existing configuration from a file"`

	All    bool     `long:"all" description:"Deploy all deployments"`
	Except []string `long:"except" description:"If used with --all, will exclude the following deployments"`
}

const ErrMsg = "Please make sure you have an existing valid kube config. You might want to check one of the following things:\n\n* Make sure you can use 'kubectl get namespaces' locally\n* If you are using Loft, you might want to run 'devspace create space' or 'loft create space'\n"

func CreateDeployments(ctx devspacecontext.Context, pipeline types.Pipeline, args []string, stdout io.Writer) error {
	ctx.Log().Debugf("create_deployments %s", strings.Join(args, " "))
	err := pipeline.Exclude(ctx)
	if err != nil {
		return err
	}
	if ctx.KubeClient() == nil {
		return errors.Errorf(ErrMsg)
	}
	options := &CreateDeploymentsOptions{
		Options: pipeline.Options().DeployOptions,
	}
	args, err = flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}

	if options.All {
		args = []string{}
		for deployment := range ctx.Config().Config().Deployments {
			if stringutil.Contains(options.Except, deployment) {
				continue
			}

			args = append(args, deployment)
			ctx, err = applySetValues(ctx, "deployments", deployment, options.Set, options.SetString, options.From, options.FromFile)
			if err != nil {
				return err
			}
		}
		if len(args) == 0 {
			return nil
		}
	} else if len(args) > 0 {
		for _, deployment := range args {
			ctx, err = applySetValues(ctx, "deployments", deployment, options.Set, options.SetString, options.From, options.FromFile)
			if err != nil {
				return err
			}

			if ctx.Config().Config().Deployments == nil || ctx.Config().Config().Deployments[deployment] == nil {
				return fmt.Errorf("couldn't find deployment %v", deployment)
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

func applySetValues(ctx devspacecontext.Context, name, objName string, set, setString, from, fromFiles []string) (devspacecontext.Context, error) {
	if len(set) == 0 && len(setString) == 0 && len(from) == 0 && len(fromFiles) == 0 {
		return ctx, nil
	}

	rawConfigOriginal := ctx.Config().RawBeforeConversion()
	rawConfig := map[string]interface{}{}
	err := util.Convert(rawConfigOriginal, &rawConfig)
	if err != nil {
		return nil, err
	}

	if rawConfig[name] == nil {
		rawConfig[name] = map[string]interface{}{}
	}
	_, ok := rawConfig[name].(map[string]interface{})
	if !ok {
		return ctx, nil
	}
	if rawConfig[name].(map[string]interface{})[objName] == nil {
		rawConfig[name].(map[string]interface{})[objName] = map[string]interface{}{}
	}

	mapObj := rawConfig[name].(map[string]interface{})[objName].(map[string]interface{})
	for _, f := range fromFiles {
		f, ok := matchesObjName(f, objName)
		if !ok {
			continue
		}

		out, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("read %s: %v", f, err)
		}

		m := map[string]interface{}{}
		err = yamlutil.Unmarshal(out, m)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling %s, %v", f, err)
		}

		mapObj = strvals.MergeMaps(mapObj, m)
	}

	for _, f := range from {
		f, ok := matchesObjName(f, objName)
		if !ok {
			continue
		}
		if rawConfig[name].(map[string]interface{})[f] == nil {
			return nil, fmt.Errorf("couldn't find --from %s", f)
		}

		mapObj = strvals.MergeMaps(mapObj, rawConfig[name].(map[string]interface{})[f].(map[string]interface{}))
	}

	for _, s := range set {
		s, ok := matchesObjName(s, objName)
		if !ok {
			continue
		}

		err = strvals.ParseInto(s, mapObj)
		if err != nil {
			return nil, errors.Wrap(err, "parsing --set flag")
		}
	}

	for _, s := range setString {
		s, ok := matchesObjName(s, objName)
		if !ok {
			continue
		}

		err = strvals.ParseIntoString(s, mapObj)
		if err != nil {
			return nil, errors.Wrap(err, "parsing --set-string flag")
		}
	}

	rawConfig[name].(map[string]interface{})[objName] = mapObj
	latestConfig, err := versions.Parse(rawConfig, ctx.Log())
	if err != nil {
		return nil, err
	}

	return ctx.WithConfig(config.NewConfig(
		ctx.Config().Raw(),
		rawConfig,
		latestConfig,
		ctx.Config().LocalCache(),
		ctx.Config().RemoteCache(),
		ctx.Config().Variables(),
		ctx.Config().Path(),
	)), nil
}

func matchesObjName(s string, objName string) (string, bool) {
	splitted := strings.Split(s, ":")
	if len(splitted) > 1 {
		if strings.Contains(splitted[0], ".") || strings.Contains(splitted[0], "=") {
			return s, true
		} else if splitted[0] != objName {
			return "", false
		}

		return strings.Join(splitted[1:], ":"), true
	}

	return s, true
}
