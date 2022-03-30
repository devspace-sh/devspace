package runtime

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/expression"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/legacy"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/kubectl/walk"
	"github.com/loft-sh/devspace/pkg/devspace/imageselector"
	varspkg "github.com/loft-sh/devspace/pkg/util/vars"
	"strings"
)

func varMatchFn(key, value string) bool {
	return true
}

// NewRuntimeResolver creates a new resolver that caches resolved variables in memory and in the provided cache
func NewRuntimeResolver(workingDir string, enableLegacyHelpers bool) RuntimeResolver {
	return &runtimeResolver{
		workingDirectory:    workingDir,
		enableLegacyHelpers: enableLegacyHelpers,
	}
}

type runtimeResolver struct {
	enableLegacyHelpers bool
	workingDirectory    string
}

func (r *runtimeResolver) FillRuntimeVariablesAsString(ctx context.Context, haystack interface{}, config config.Config, dependencies []types.Dependency) (string, error) {
	out, err := r.FillRuntimeVariables(ctx, haystack, config, dependencies)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", out), nil
}

func (r *runtimeResolver) FillRuntimeVariablesAsImageSelector(ctx context.Context, haystack interface{}, config config.Config, dependencies []types.Dependency) (*imageselector.ImageSelector, error) {
	out, err := r.FillRuntimeVariablesAsString(ctx, haystack, config, dependencies)
	if err != nil {
		return nil, err
	}

	return &imageselector.ImageSelector{
		Image: out,
	}, nil
}

func (r *runtimeResolver) FillRuntimeVariablesWithRebuild(ctx context.Context, haystack interface{}, config config.Config, dependencies []types.Dependency) (bool, interface{}, error) {
	shouldRebuild, haystack, err := r.fillVariables(haystack, config, dependencies, r.enableLegacyHelpers)
	if err != nil {
		return false, nil, err
	}

	// resolve expressions
	haystack, err = expression.ResolveAllExpressions(ctx, haystack, r.workingDirectory, nil, nil, config.Variables())
	if err != nil {
		return false, nil, err
	}

	// just resolve variables again
	rebuild, haystack, err := r.fillVariables(haystack, config, dependencies, false)
	if err != nil {
		return false, nil, err
	}

	return shouldRebuild || rebuild, haystack, nil
}

func (r *runtimeResolver) FillRuntimeVariables(ctx context.Context, haystack interface{}, config config.Config, dependencies []types.Dependency) (interface{}, error) {
	_, out, err := r.FillRuntimeVariablesWithRebuild(ctx, haystack, config, dependencies)
	return out, err
}

func (r *runtimeResolver) fillVariables(haystack interface{}, config config.Config, dependencies []types.Dependency, legacyHelpers bool) (bool, interface{}, error) {
	switch t := haystack.(type) {
	case string:
		return r.replaceString(t, config, dependencies, legacyHelpers)
	case map[string]interface{}:
		shouldRebuild := false
		err := walk.Walk(t, varMatchFn, func(path, value string) (interface{}, error) {
			rebuild, val, err := r.replaceString(value, config, dependencies, legacyHelpers)
			shouldRebuild = shouldRebuild || rebuild
			return val, err
		})
		return shouldRebuild, t, err
	}

	return false, nil, fmt.Errorf("unrecognized haystack type: %#v", haystack)
}

func (r *runtimeResolver) replaceString(str string, config config.Config, dependencies []types.Dependency, legacyHelpers bool) (bool, interface{}, error) {
	shouldRebuild := false
	value, err := varspkg.ParseString(str, func(name string) (interface{}, error) {
		if strings.HasPrefix(name, "runtime.") {
			return "${" + name + "}", nil
		}

		rebuild, val, err := r.resolve(name, config, dependencies)
		if err != nil {
			return "", err
		}

		shouldRebuild = shouldRebuild || rebuild
		return val, nil
	})
	if err != nil {
		return false, nil, err
	}

	valueStr, ok := value.(string)
	if !ok {
		return shouldRebuild, value, nil
	} else {
		str = valueStr
	}

	if legacyHelpers {
		rebuild, val, err := legacy.Replace(str, config, dependencies)
		if err != nil {
			return false, "", err
		}

		shouldRebuild = shouldRebuild || rebuild
		str = fmt.Sprintf("%v", val)
	}

	value, err = varspkg.ParseString(str, func(name string) (interface{}, error) {
		if !strings.HasPrefix(name, "runtime.") {
			return "${" + name + "}", nil
		}

		rebuild, val, err := r.resolve(name, config, dependencies)
		if err != nil {
			return "", err
		}

		shouldRebuild = shouldRebuild || rebuild
		return val, nil
	})
	return shouldRebuild, value, err
}

func (r *runtimeResolver) resolve(name string, config config.Config, dependencies []types.Dependency) (bool, interface{}, error) {
	name = strings.TrimSpace(name)

	// check if in vars already
	v, ok := config.Variables()[name]
	if ok {
		return false, v, nil
	}

	// fill the variable if not found
	shouldRebuild, value, err := r.fillRuntimeVariable(name, config, dependencies)
	if err != nil {
		return false, nil, err
	}

	return shouldRebuild, value, nil
}

func (r *runtimeResolver) fillRuntimeVariable(name string, config config.Config, dependencies []types.Dependency) (bool, interface{}, error) {
	// is runtime variable
	if strings.HasPrefix(name, "runtime.") {
		return NewRuntimeVariable(name, config, dependencies).Load()
	}

	return false, "${" + name + "}", nil
}
