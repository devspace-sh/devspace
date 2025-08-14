package commands

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/loft-sh/devspace/pkg/util/yamlutil"
	"github.com/pkg/errors"
	"strings"

	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
	"mvdan.cc/sh/v3/interp"
)

type GetConfigValueOptions struct{}

func GetConfigValue(ctx devspacecontext.Context, args []string) error {
	ctx = ctx.WithLogger(ctx.Log().ErrorStreamOnly())
	ctx.Log().Debugf("get_config_value %s", strings.Join(args, " "))
	options := &GetConfigValueOptions{}
	args, err := flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	} else if len(args) != 1 {
		return fmt.Errorf("usage: get_config_value deployments.my-deployment.helm.chart.name")
	}

	hc := interp.HandlerCtx(ctx.Context())
	config := ctx.Config().RawBeforeConversion()
	nodePath, err := yamlpath.NewPath(args[0])
	if err != nil {
		ctx.Log().Debugf("%v", err)
		return nil
	}

	out, err := yaml.Marshal(config)
	if err != nil {
		ctx.Log().Debugf("%v", err)
		return nil
	}

	var doc yaml.Node
	err = yamlutil.Unmarshal(out, &doc)
	if err != nil {
		ctx.Log().Debugf("%v", err)
		return nil
	}

	nodes, err := nodePath.Find(&doc)
	if err != nil {
		ctx.Log().Debugf("%v", err)
		return nil
	}
	if len(nodes) < 1 {
		return nil
	}

	if nodes[0].Kind == yaml.ScalarNode {
		_, _ = hc.Stdout.Write([]byte(nodes[0].Value))
		return nil
	}

	out, err = yaml.Marshal(nodes[0])
	if err != nil {
		return err
	}
	_, _ = hc.Stdout.Write(out)
	return nil
}
