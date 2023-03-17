package variable

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/expression"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/graph"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/kubectl/walk"
	"github.com/loft-sh/devspace/pkg/util/log"
	varspkg "github.com/loft-sh/devspace/pkg/util/vars"
	"github.com/pkg/errors"
)

var AlwaysResolvePredefinedVars = []string{"DEVSPACE_NAME", "DEVSPACE_EXECUTABLE", "DEVSPACE_KUBECTL_EXECUTABLE", "DEVSPACE_TMPDIR", "DEVSPACE_VERSION", "DEVSPACE_RANDOM", "DEVSPACE_PROFILE", "DEVSPACE_PROFILES", "DEVSPACE_USER_HOME", "DEVSPACE_TIMESTAMP", "devspace.context", "DEVSPACE_CONTEXT", "devspace.namespace", "DEVSPACE_NAMESPACE", "DEVSPACE_SPACE"}

// NewResolver creates a new resolver that caches resolved variables in memory and in the provided cache
func NewResolver(localCache localcache.Cache, predefinedVariableOptions *PredefinedVariableOptions, flags []string, log log.Logger) (Resolver, error) {
	memoryCache := map[string]interface{}{}
	err := MergeVarsWithFlags(memoryCache, flags)
	if err != nil {
		return nil, err
	}

	return &resolver{
		memoryCache: memoryCache,
		localCache:  localCache,
		options:     predefinedVariableOptions,
		log:         log,
	}, nil
}

func MergeVarsWithFlags(vars map[string]interface{}, flags []string) error {
	for _, cmdVar := range flags {
		idx := strings.Index(cmdVar, "=")
		if idx == -1 {
			return errors.Errorf("wrong --var format: %s, expected 'key=val'", cmdVar)
		}

		name := strings.TrimSpace(cmdVar[:idx])
		value := convertStringValue(strings.TrimSpace(cmdVar[idx+1:]))
		vars[name] = value
	}

	return nil
}

type resolver struct {
	vars        map[string]*latest.Variable
	memoryCache map[string]interface{}

	localCache localcache.Cache
	options    *PredefinedVariableOptions
	log        log.Logger
}

func varMatchFn(key, value string) bool {
	return varspkg.VarMatchRegex.MatchString(value)
}

func (r *resolver) DefinedVars() map[string]*latest.Variable {
	return r.vars
}

func (r *resolver) UpdateVars(vars map[string]*latest.Variable) {
	r.vars = vars
}

func (r *resolver) fillVariables(ctx context.Context, haystack interface{}, exclude, include []*regexp.Regexp) (interface{}, error) {
	switch t := haystack.(type) {
	case string:
		return r.replaceString(ctx, t)
	case map[string]interface{}:
		err := walk.Walk(t, varMatchFn, func(path, value string) (interface{}, error) {
			if expression.ExcludedPath(path, exclude, include) {
				return value, nil
			}

			return r.replaceString(ctx, value)
		})
		return t, err
	case []interface{}:
		for i := range t {
			var err error
			t[i], err = r.fillVariables(ctx, t[i], exclude, include)
			if err != nil {
				return nil, err
			}
		}

		return t, nil
	}

	return haystack, nil
}

func (r *resolver) ResolvedVariables() map[string]interface{} {
	return r.memoryCache
}

func (r *resolver) replaceString(ctx context.Context, str string) (interface{}, error) {
	return varspkg.ParseString(str, func(v string) (interface{}, error) {
		val, err := r.resolve(ctx, v, nil)
		if err != nil {
			return "", err
		}

		return val, nil
	})
}

func (r *resolver) findVariables(haystack interface{}, skipUnused bool, include []*regexp.Regexp) ([]*latest.Variable, error) {
	// find out what vars are really used
	varsUsed := map[string]bool{}
	switch t := haystack.(type) {
	case string:
		_, _ = varspkg.ParseString(t, func(v string) (interface{}, error) {
			varsUsed[v] = true
			return "", nil
		})
	case map[string]interface{}:
		err := walk.Walk(t, varMatchFn, func(path, value string) (interface{}, error) {
			if expression.ExcludedPath(path, nil, include) {
				return value, nil
			}

			_, _ = varspkg.ParseString(value, func(v string) (interface{}, error) {
				varsUsed[v] = true
				return "", nil
			})

			return value, nil
		})
		if err != nil {
			return nil, err
		}
	}

	// add always resolve variables. If skip unused is true, we don't
	// resolve by default and instead only resolve the actually found variables
	if !skipUnused {
		for _, v := range r.vars {
			if v.AlwaysResolve == nil || *v.AlwaysResolve {
				varsUsed[v.Name] = true
			}
		}
	}

	// filter out runtime environment variables
	for k := range varsUsed {
		if !strings.HasPrefix(k, "runtime.") && !IsPredefinedVariable(k) && r.getVariableDefinition(k) == nil {
			delete(varsUsed, k)
		}
	}

	return r.orderVariables(varsUsed)
}

func (r *resolver) FindVariables(haystack interface{}) ([]*latest.Variable, error) {
	return r.findVariables(haystack, false, nil)
}

func (r *resolver) getVariableDefinition(name string) *latest.Variable {
	definition, ok := r.vars[name]
	if !ok {
		value := os.Getenv(name)
		if value != "" {
			return &latest.Variable{
				Name:  name,
				Value: value,
			}
		}

		return nil
	}

	return definition
}

func (r *resolver) orderVariables(vars map[string]bool) ([]*latest.Variable, error) {
	root := graph.NewNode("root", nil)
	g := graph.NewGraphOf(root, "variable")
	for name := range vars {
		// check if predefined variable
		var definition *latest.Variable
		if IsPredefinedVariable(name) {
			definition = &latest.Variable{Name: name}
		} else {
			// check if has definition
			definition = r.getVariableDefinition(name)
			if definition == nil {
				continue
			}
		}

		err := r.insertVariableGraph(g, definition)
		if err != nil {
			return nil, err
		}
	}

	// now get all the leaf nodes
	retVars := []*latest.Variable{}
	for {
		nextLeaf := g.GetNextLeaf(root)
		if nextLeaf == root {
			break
		}

		retVars = append(retVars, nextLeaf.Data.(*latest.Variable))
		err := g.RemoveNode(nextLeaf.ID)
		if err != nil {
			return nil, err
		}
	}

	// reverse the slice
	for i, j := 0, len(retVars)-1; i < j; i, j = i+1, j-1 {
		retVars[i], retVars[j] = retVars[j], retVars[i]
	}

	return retVars, nil
}

func (r *resolver) insertVariableGraph(g *graph.Graph, node *latest.Variable) error {
	if _, ok := g.Nodes[node.Name]; !ok {
		_, err := g.InsertNodeAt("root", node.Name, node)
		if err != nil {
			return err
		}
	}

	parents := r.findVariablesInDefinition(node)
	for parent := range parents {
		parentDefinition := r.getVariableDefinition(parent)
		if parentDefinition == nil {
			continue
		}

		if _, ok := g.Nodes[parentDefinition.Name]; !ok {
			err := r.insertVariableGraph(g, parentDefinition)
			if err != nil {
				return err
			}
		}

		err := g.AddEdge(parent, node.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *resolver) FillVariablesInclude(ctx context.Context, haystack interface{}, skipUnused bool, includedPaths []string) (interface{}, error) {
	paths := []*regexp.Regexp{}
	for _, path := range includedPaths {
		path = strings.ReplaceAll(path, "**", ".+")
		path = strings.ReplaceAll(path, "*", "[^/]+")
		path = "^" + path + "$"
		expr, err := regexp.Compile(path)
		if err != nil {
			return nil, err
		}

		paths = append(paths, expr)
	}

	// fill variables
	preparedConfigInterface, err := r.findAndFillVariables(ctx, haystack, skipUnused, nil, paths)
	if err != nil {
		return nil, err
	}

	// resolve expressions
	preparedConfigInterface, err = expression.ResolveAllExpressions(ctx, preparedConfigInterface, filepath.Dir(r.options.ConfigPath), nil, paths, r.memoryCache)
	if err != nil {
		return nil, err
	}

	// fill in variables again
	return r.findAndFillVariables(ctx, preparedConfigInterface, skipUnused, nil, paths)
}

func (r *resolver) FillVariablesExclude(ctx context.Context, haystack interface{}, skipUnused bool, excludedPaths []string) (interface{}, error) {
	paths := []*regexp.Regexp{}
	for _, path := range excludedPaths {
		path = strings.ReplaceAll(path, "**", ".+")
		path = strings.ReplaceAll(path, "*", "[^/]+")
		path = "^" + path + "$"
		expr, err := regexp.Compile(path)
		if err != nil {
			return nil, err
		}

		paths = append(paths, expr)
	}

	// fill variables
	preparedConfigInterface, err := r.findAndFillVariables(ctx, haystack, skipUnused, paths, nil)
	if err != nil {
		return nil, err
	}

	// resolve expressions
	preparedConfigInterface, err = expression.ResolveAllExpressions(ctx, preparedConfigInterface, filepath.Dir(r.options.ConfigPath), paths, nil, r.memoryCache)
	if err != nil {
		return nil, err
	}

	// fill in variables again
	return r.findAndFillVariables(ctx, preparedConfigInterface, skipUnused, paths, nil)
}

func (r *resolver) FillVariables(ctx context.Context, haystack interface{}, skipUnused bool) (interface{}, error) {
	return r.FillVariablesExclude(ctx, haystack, skipUnused, nil)
}

func (r *resolver) findAndFillVariables(ctx context.Context, haystack interface{}, skipUnused bool, exclude, include []*regexp.Regexp) (interface{}, error) {
	varsUsed, err := r.findVariables(haystack, skipUnused, include)
	if err != nil {
		return nil, err
	}

	// try resolving predefined variables
	for _, name := range AlwaysResolvePredefinedVars {
		// ignore errors here as those variables are probably not used anyways
		_, err := r.resolve(ctx, name, nil)
		if err != nil {
			r.log.Debugf("error resolving predefined variable: %v", err)
		}
	}

	// resolve used defined variables
	for _, v := range varsUsed {
		_, err := r.resolve(ctx, v.Name, v)
		if err != nil {
			return nil, err
		}
	}

	return r.fillVariables(ctx, haystack, exclude, include)
}

func (r *resolver) resolve(ctx context.Context, name string, definition *latest.Variable) (interface{}, error) {
	name = strings.TrimSpace(name)

	// check if in vars already
	v, ok := r.memoryCache[name]
	if ok {
		return v, nil
	}

	// is predefined variable?
	variable, err := NewPredefinedVariable(name, r.options, r.log)
	if err == nil {
		value, err := variable.Load(ctx, definition)
		if err != nil {
			return nil, err
		}

		r.memoryCache[name] = value
		return value, nil
	}

	// fill other variables in the variable definition
	err = r.fillVariableDefinition(ctx, definition)
	if err != nil {
		return nil, err
	}

	// skip variable if no definition was found
	if definition == nil {
		return "${" + name + "}", nil
	}

	// fill the variable if not found
	value, err := r.fillVariable(ctx, name, definition)
	if err != nil {
		return nil, err
	}

	// set variable so that we don't ask again
	r.memoryCache[name] = value
	return value, nil
}

func (r *resolver) findVariablesInDefinition(definition *latest.Variable) map[string]bool {
	varsUsed := map[string]bool{}
	if definition == nil {
		return varsUsed
	}

	// check value
	if strDefault, ok := definition.Value.(string); ok {
		_, _ = varspkg.ParseString(strDefault, func(v string) (interface{}, error) {
			varsUsed[v] = true
			return "", nil
		})
	}

	// check default value
	if strDefault, ok := definition.Default.(string); ok {
		_, _ = varspkg.ParseString(strDefault, func(v string) (interface{}, error) {
			varsUsed[v] = true
			return "", nil
		})
	}

	// check command
	_, _ = varspkg.ParseString(definition.Command, func(v string) (interface{}, error) {
		varsUsed[v] = true
		return "", nil
	})

	// check args
	for _, arg := range definition.Args {
		_, _ = varspkg.ParseString(arg, func(v string) (interface{}, error) {
			varsUsed[v] = true
			return "", nil
		})
	}

	// check commands
	for _, osDef := range definition.Commands {
		// check command
		_, _ = varspkg.ParseString(osDef.Command, func(v string) (interface{}, error) {
			varsUsed[v] = true
			return "", nil
		})

		// check args
		for _, arg := range osDef.Args {
			_, _ = varspkg.ParseString(arg, func(v string) (interface{}, error) {
				varsUsed[v] = true
				return "", nil
			})
		}
	}

	// filter out runtime environment variables and non existing ones
	for k := range varsUsed {
		if !strings.HasPrefix(k, "runtime.") && !IsPredefinedVariable(k) && r.getVariableDefinition(k) == nil {
			delete(varsUsed, k)
		}
	}

	return varsUsed
}

func (r *resolver) fillVariableDefinition(ctx context.Context, definition *latest.Variable) error {
	var err error
	if definition == nil {
		return nil
	}

	// this converts the definition.Value to definition.Default
	if definition.Value != nil {
		if definition.Default != nil {
			return fmt.Errorf(".default cannot be used with .value together for variable ${%s}", definition.Name)
		}

		definition.Default = definition.Value
		definition.Source = latest.VariableSourceNone
	}

	// if the definition has a default value, we try to resolve possible variables
	// in that definition from the cache (or predefined) before continuing
	if definition.Default != nil {
		resolvedDefaultValue, err := r.resolveDefaultValue(ctx, definition)
		if err != nil {
			return err
		}

		definition.Default = resolvedDefaultValue
	}

	// resolve command
	definition.Command, err = r.resolveDefinitionStringToString(ctx, definition.Command, definition)
	if err != nil {
		return err
	}

	// resolve args
	for i := range definition.Args {
		definition.Args[i], err = r.resolveDefinitionStringToString(ctx, definition.Args[i], definition)
		if err != nil {
			return err
		}
	}

	// resolve commands
	for ci := range definition.Commands {
		definition.Commands[ci].Command, err = r.resolveDefinitionStringToString(ctx, definition.Commands[ci].Command, definition)
		if err != nil {
			return err
		}
		for i := range definition.Commands[ci].Args {
			definition.Commands[ci].Args[i], err = r.resolveDefinitionStringToString(ctx, definition.Commands[ci].Args[i], definition)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *resolver) resolveDefinitionStringToString(ctx context.Context, str string, definition *latest.Variable) (string, error) {
	val, err := r.resolveDefinitionString(ctx, str, definition)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", val), nil
}

func (r *resolver) resolveDefinitionString(ctx context.Context, str string, definition *latest.Variable) (interface{}, error) {
	return varspkg.ParseString(str, func(varName string) (interface{}, error) {
		v, ok := r.memoryCache[varName]
		if !ok {
			// check if its a predefined variable
			variable, err := NewPredefinedVariable(varName, r.options, r.log)
			if err != nil {
				if r.getVariableDefinition(varName) == nil {
					return "${" + varName + "}", nil
				}

				return nil, errors.Errorf("variable '%s' was not resolved yet, however is used in the definition of variable '%s' as '%s'. Please make sure you define '%s' before '%s' in the vars array", varName, definition.Name, str, varName, definition.Name)
			}

			return variable.Load(ctx, definition)
		}

		return v, nil
	})
}

func (r *resolver) resolveDefaultValue(ctx context.Context, definition *latest.Variable) (interface{}, error) {
	// check if default value is a string
	defaultString, ok := definition.Default.(string)
	if !ok {
		return definition.Default, nil
	}

	return r.resolveDefinitionString(ctx, defaultString, definition)
}

func (r *resolver) fillVariable(ctx context.Context, name string, definition *latest.Variable) (interface{}, error) {
	// is runtime variable
	if strings.HasPrefix(name, "runtime.") {
		return nil, fmt.Errorf("cannot resolve %s in this config area as this config region is loaded on startup. You can only use runtime variables in the following locations: \n  %s", name, strings.Join(runtime.Locations, "\n  "))
	}

	// fill variable without definition
	if definition == nil {
		return NewUndefinedVariable(name, r.localCache, r.log).Load(ctx, definition)
	}

	// trim space from variable definition
	definition.Name = strings.TrimSpace(definition.Name)

	// fill variable by source
	switch definition.Source {
	case latest.VariableSourceEnv:
		return NewEnvVariable(name).Load(ctx, definition)
	case latest.VariableSourceDefault, latest.VariableSourceInput, latest.VariableSourceAll:
		return NewDefaultVariable(name, filepath.Dir(r.options.ConfigPath), r.localCache, r.log).Load(ctx, definition)
	case latest.VariableSourceNone:
		return NewNoneVariable(name).Load(ctx, definition)
	case latest.VariableSourceCommand:
		return NewCommandVariable(name, filepath.Dir(r.options.ConfigPath)).Load(ctx, definition)
	default:
		return nil, errors.Errorf("unrecognized variable source '%s', please choose one of 'all', 'input', 'env' or 'none'", name)
	}
}
