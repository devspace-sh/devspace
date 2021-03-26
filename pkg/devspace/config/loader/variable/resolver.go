package variable

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/kubectl/walk"
	"github.com/loft-sh/devspace/pkg/util/log"
	varspkg "github.com/loft-sh/devspace/pkg/util/vars"
	"github.com/pkg/errors"
	"strings"
)

// NewResolver creates a new resolver that caches resolved variables in memory and in the provided cache
func NewResolver(cache map[string]string, predefinedVariableOptions *PredefinedVariableOptions, log log.Logger) Resolver {
	return &resolver{
		memoryCache:     map[string]interface{}{},
		persistentCache: cache,
		options:         predefinedVariableOptions,
		log:             log,
	}
}

type resolver struct {
	memoryCache     map[string]interface{}
	persistentCache map[string]string
	options         *PredefinedVariableOptions
	log             log.Logger
}

func varMatchFn(key, value string) bool {
	return varspkg.VarMatchRegex.MatchString(value)
}

func (r *resolver) FillVariables(haystack map[interface{}]interface{}) error {
	err := walk.Walk(haystack, varMatchFn, func(value string) (interface{}, error) {
		return r.ReplaceString(value)
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *resolver) ResolvedVariables() map[string]interface{} {
	return r.memoryCache
}

func (r *resolver) ReplaceString(str string) (interface{}, error) {
	return varspkg.ParseString(str, func(v string) (interface{}, error) {
		val, err := r.Resolve(v, nil)
		if err != nil {
			return "", err
		}

		return val, nil
	})
}

func (r *resolver) FindVariables(haystack map[interface{}]interface{}, vars []*latest.Variable) (map[string]bool, error) {
	// find out what vars are really used
	varsUsed := map[string]bool{}
	err := walk.Walk(haystack, varMatchFn, func(value string) (interface{}, error) {
		_, _ = varspkg.ParseString(value, func(v string) (interface{}, error) {
			varsUsed[v] = true
			return "", nil
		})

		return value, nil
	})
	if err != nil {
		return nil, err
	}

	// find out what vars are used within other vars default values
	for _, v := range vars {
		if strDefault, ok := v.Default.(string); ok {
			_, _ = varspkg.ParseString(strDefault, func(v string) (interface{}, error) {
				varsUsed[v] = true
				return "", nil
			})
		}
	}

	return varsUsed, nil
}

func (r *resolver) ConvertFlags(flags []string) (map[string]interface{}, error) {
	retVariables := map[string]interface{}{}
	for _, cmdVar := range flags {
		idx := strings.Index(cmdVar, "=")
		if idx == -1 {
			return nil, errors.Errorf("wrong --var format: %s, expected 'key=val'", cmdVar)
		}

		name := strings.TrimSpace(cmdVar[:idx])
		value := convertStringValue(strings.TrimSpace(cmdVar[idx+1:]))
		r.memoryCache[name] = value
		retVariables[name] = value
	}

	return retVariables, nil
}

func (r *resolver) Resolve(name string, definition *latest.Variable) (interface{}, error) {
	name = strings.TrimSpace(name)

	// check if in vars already
	v, ok := r.memoryCache[name]
	if ok {
		return v, nil
	}

	// if the definition has a default value, we try to resolve possible variables
	// in that definition from the cache (or predefined) before continuing
	if definition != nil && definition.Default != nil {
		resolvedDefaultValue, err := r.resolveDefaultValue(definition)
		if err != nil {
			return nil, err
		}

		definition.Default = resolvedDefaultValue
	}

	// fill the variable if not found
	value, err := r.fillVariable(name, definition)
	if err != nil {
		return nil, err
	}

	// set variable so that we don't ask again
	r.memoryCache[name] = value
	return value, nil
}

func (r *resolver) resolveDefaultValue(definition *latest.Variable) (interface{}, error) {
	// check if default value is a string
	defaultString, ok := definition.Default.(string)
	if !ok {
		return definition.Default, nil
	}

	return varspkg.ParseString(defaultString, func(varName string) (interface{}, error) {
		v, ok := r.memoryCache[varName]
		if !ok {
			// check if its a predefined variable
			variable, err := NewPredefinedVariable(varName, r.persistentCache, r.options)
			if err != nil {
				return nil, errors.Errorf("variable %s was not resolved yet, however is used in the default value of variable %s. Please make sure you define %s before %s", varName, definition.Name, varName, definition.Name)
			}

			return variable.Load(definition)
		}

		return v, nil
	})
}

func (r *resolver) fillVariable(name string, definition *latest.Variable) (interface{}, error) {
	// is predefined variable?
	variable, err := NewPredefinedVariable(name, r.persistentCache, r.options)
	if err == nil {
		return variable.Load(definition)
	}

	// fill variable without definition
	if definition == nil {
		return NewUndefinedVariable(name, r.persistentCache, r.log).Load(definition)
	}

	// trim space from variable definition
	definition.Name = strings.TrimSpace(definition.Name)

	// fill variable by source
	switch definition.Source {
	case latest.VariableSourceEnv:
		return NewEnvVariable(name).Load(definition)
	case latest.VariableSourceDefault, latest.VariableSourceInput, latest.VariableSourceAll:
		return NewDefaultVariable(name, r.persistentCache, r.log).Load(definition)
	case latest.VariableSourceNone:
		return NewNoneVariable(name).Load(definition)
	case latest.VariableSourceCommand:
		return NewCommandVariable(name).Load(definition)
	default:
		return nil, errors.Errorf("unrecognized variable source '%s', please choose one of 'all', 'input', 'env' or 'none'", name)
	}
}
