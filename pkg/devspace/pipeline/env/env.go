package env

import (
	"fmt"
	"os"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	enginetypes "github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/types"
	"mvdan.cc/sh/v3/expand"
)

var _ expand.Environ = &envProvider{}

func NewVariableEnvProvider(config config.Config, dependencies []types.Dependency, extraEnvVars map[string]string) expand.Environ {
	env := os.Environ()
	for k, v := range extraEnvVars {
		env = append(env, k+"="+v)
	}

	return &envProvider{
		listProvider: expand.ListEnviron(env...),
		config:       config,
		dependencies: dependencies,
	}
}

type envProvider struct {
	listProvider expand.Environ
	config       config.Config
	dependencies []types.Dependency
	// workingDir   string
}

func (e *envProvider) Get(name string) expand.Variable {
	// Should we enable this?
	// v, ok := e.getRuntimeVariable(name)
	// if ok {
	//	return v
	// }

	v, ok := e.getVariable(name)
	if ok {
		return v
	}

	return e.listProvider.Get(name)
}

func (e *envProvider) getVariable(name string) (expand.Variable, bool) {
	replacedName := strings.ReplaceAll(name, enginetypes.DotReplacement, ".")
	v, ok := e.config.Variables()[replacedName]
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

// func (e *envProvider) getRuntimeVariable(name string) (expand.Variable, bool) {
// 	replacedName := strings.ReplaceAll(name, enginetypes.DotReplacement, ".")
// 	_, val, err := runtime.NewRuntimeVariable(replacedName, e.config, e.dependencies).Load()
// 	if err != nil {
// 		return expand.Variable{}, false
// 	} else if val != nil {
// 		return expand.Variable{
// 			Exported: true,
// 			ReadOnly: true,
// 			Kind:     expand.String,
// 			Str:      fmt.Sprintf("%v", val),
// 		}, true
// 	}

// 	return expand.Variable{}, false
// }

func (e *envProvider) Each(visitor func(name string, vr expand.Variable) bool) {
	// Should we enable this?
	// for name := range e.config.ListRuntimeVariables() {
	//	v, ok := e.getRuntimeVariable(name)
	//	if ok {
	//		visitor(name, v)
	//	}
	// }

	for name := range e.config.Variables() {
		v, ok := e.getVariable(name)
		if ok {
			visitor(name, v)
		}
	}

	e.listProvider.Each(visitor)
}
