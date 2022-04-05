package env

import (
	"fmt"
	enginetypes "github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/types"
	"mvdan.cc/sh/v3/expand"
	"os"
	"strings"
	"sync"
)

type Provider interface {
	expand.Environ
	Set(envVars map[string]string)
}

func NewVariableEnvProvider(envVars map[string]string) Provider {
	p := &provider{}
	p.Set(envVars)
	return p
}

type provider struct {
	m sync.Mutex

	listProvider expand.Environ
}

func (p *provider) Set(envVars map[string]string) {
	p.m.Lock()
	defer p.m.Unlock()

	env := os.Environ()
	for k, v := range envVars {
		key := strings.ReplaceAll(k, enginetypes.DotReplacement, ".")
		env = append(env, key+"="+v)
	}
	p.listProvider = expand.ListEnviron(env...)
}

func (p *provider) Get(name string) expand.Variable {
	p.m.Lock()
	defer p.m.Unlock()

	name = strings.ReplaceAll(name, enginetypes.DotReplacement, ".")
	return p.listProvider.Get(name)
}

func (p *provider) Each(visitor func(name string, vr expand.Variable) bool) {
	p.m.Lock()
	defer p.m.Unlock()

	p.listProvider.Each(visitor)
}

func ConvertMap(iMap map[string]interface{}) map[string]string {
	retMap := map[string]string{}
	for k, v := range iMap {
		retMap[k] = fmt.Sprintf("%v", v)
	}
	return retMap
}
