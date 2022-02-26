package env

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"mvdan.cc/sh/v3/expand"
	"os"
)

var _ expand.Environ = &envProvider{}

func NewVariableEnvProvider(config config.Config, extraEnvVars map[string]string) expand.Environ {
	env := os.Environ()
	for k, v := range extraEnvVars {
		env = append(env, k+"="+v)
	}

	return &envProvider{
		listProvider: expand.ListEnviron(env...),
		config:       config,
	}
}

type envProvider struct {
	listProvider expand.Environ
	config       config.Config
}

func (e *envProvider) Get(name string) expand.Variable {
	v, ok := e.getRuntimeVariable(name)
	if ok {
		return v
	}

	v, ok = e.getVariable(name)
	if ok {
		return v
	}

	return e.listProvider.Get(name)
}

func (e *envProvider) getVariable(name string) (expand.Variable, bool) {
	v, ok := e.config.Variables()[name]
	if ok {
		return expand.Variable{
			Exported: true,
			ReadOnly: true,
			Kind:     expand.String,
			Str:      fmt.Sprintf("%v", v),
		}, true
	}

	return expand.Variable{}, false
}

func (e *envProvider) getRuntimeVariable(name string) (expand.Variable, bool) {
	value, ok := e.config.GetRuntimeVariable(name)
	if ok {
		return expand.Variable{
			Exported: true,
			ReadOnly: true,
			Kind:     expand.String,
			Str:      fmt.Sprintf("%v", value),
		}, true
	}

	return expand.Variable{}, false
}

func (e *envProvider) Each(visitor func(name string, vr expand.Variable) bool) {
	for name := range e.config.ListRuntimeVariables() {
		v, ok := e.getRuntimeVariable(name)
		if ok {
			visitor(name, v)
		}
	}

	for name := range e.config.Variables() {
		v, ok := e.getVariable(name)
		if ok {
			visitor(name, v)
		}
	}

	e.listProvider.Each(visitor)
}
