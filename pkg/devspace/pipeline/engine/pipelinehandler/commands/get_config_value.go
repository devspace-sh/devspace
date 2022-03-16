package commands

import (
	"fmt"
	"strings"

	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
	"mvdan.cc/sh/v3/interp"
)

func GetConfigValue(ctx *devspacecontext.Context, args []string) error {
	ctx.Log.Debugf("get_config_value %s", strings.Join(args, " "))
	if len(args) != 1 {
		return fmt.Errorf("usage: get_config_value deployments.my-deployment.helm.chart.name")
	}

	hc := interp.HandlerCtx(ctx.Context)
	rawConfig := ctx.Config.Raw()

	nodePath, err := yamlpath.NewPath(args[0])
	if err != nil {
		_, _ = hc.Stderr.Write([]byte(err.Error()))
		return nil
	}

	out, err := yaml.Marshal(rawConfig)
	if err != nil {
		_, _ = hc.Stderr.Write([]byte(err.Error()))
		return nil
	}

	var doc yaml.Node
	err = yaml.Unmarshal(out, &doc)
	if err != nil {
		_, _ = hc.Stderr.Write([]byte(err.Error()))
		return nil
	}

	nodes, err := nodePath.Find(&doc)
	if err != nil {
		_, _ = hc.Stderr.Write([]byte(err.Error()))
		return nil
	}

	if len(nodes) < 1 {
		_, _ = hc.Stderr.Write([]byte("no nodes found"))
		return nil
	}

	_, _ = hc.Stdout.Write([]byte(nodes[0].Value))
	return nil
}
