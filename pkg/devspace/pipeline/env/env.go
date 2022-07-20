package env

import (
	"fmt"
	enginetypes "github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/types"
	"mvdan.cc/sh/v3/expand"
	"strings"
)

type Provider interface {
	expand.Environ
}

func NewVariableEnvProvider(base expand.Environ, envVars map[string]string) Provider {
	additionalVars := map[string]string{}
	for k, v := range envVars {
		additionalVars[strings.ReplaceAll(k, enginetypes.DotReplacement, ".")] = v
	}

	return &provider{
		base:           base,
		additionalVars: envVars,
	}
}

type provider struct {
	base           expand.Environ
	additionalVars map[string]string
}

func (p *provider) Get(name string) expand.Variable {
	name = strings.ReplaceAll(name, enginetypes.DotReplacement, ".")
	value, ok := p.additionalVars[name]
	if ok {
		return expand.Variable{
			Exported: true,
			Kind:     expand.String,
			Str:      value,
		}
	}

	return p.base.Get(name)
}

func (p *provider) Each(visitor func(name string, vr expand.Variable) bool) {
	for k, v := range p.additionalVars {
		visitor(k, expand.Variable{
			Exported: true,
			Kind:     expand.String,
			Str:      v,
		})
	}

	p.base.Each(func(name string, vr expand.Variable) bool {
		_, ok := p.additionalVars[name]
		if !ok {
			return visitor(name, vr)
		}

		return true
	})
}

func ConvertMap(iMap map[string]interface{}) map[string]string {
	retMap := map[string]string{}
	for k, v := range iMap {
		retMap[k] = fmt.Sprintf("%v", v)
	}
	return retMap
}
