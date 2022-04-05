package env

import (
	"fmt"
	"mvdan.cc/sh/v3/expand"
	"os"
	"sync"
)

type Provider interface {
	expand.Environ
	Set(envVars map[string]string)
}

func NewVariableEnvProvider(envVars map[string]string) Provider {
	env := os.Environ()
	for k, v := range envVars {
		env = append(env, k+"="+v)
	}

	return &provider{
		listProvider: expand.ListEnviron(env...),
	}
}

type provider struct {
	m sync.Mutex

	listProvider expand.Environ
	workingDir   string
}

func (p *provider) Set(envVars map[string]string) {
	p.m.Lock()
	defer p.m.Unlock()

	env := os.Environ()
	for k, v := range envVars {
		env = append(env, k+"="+v)
	}
	p.listProvider = expand.ListEnviron()
}

func (p *provider) Get(name string) expand.Variable {
	p.m.Lock()
	defer p.m.Unlock()

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
