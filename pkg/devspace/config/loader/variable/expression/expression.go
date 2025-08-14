package expression

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/env"
	"github.com/loft-sh/devspace/pkg/util/yamlutil"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"

	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/kubectl/walk"
)

// ExpressionMatchRegex is the regex to check if a value matches the devspace var format
var ExpressionMatchRegex = regexp.MustCompile(`(?ms)^\$\$?\#?\!?\((.+)\)$`)

const DevSpaceSkipPreloadEnv = "DEVSPACE_SKIP_PRELOAD"

func expressionMatchFn(key, value string) bool {
	return ExpressionMatchRegex.MatchString(value)
}

func ExcludedPath(path string, excluded, included []*regexp.Regexp) bool {
	for _, expr := range excluded {
		if expr.MatchString(path) {
			return true
		}
	}
	if len(included) > 0 {
		for _, expr := range included {
			if expr.MatchString(path) {
				return false
			}
		}

		return true
	}

	return false
}

func ResolveAllExpressions(ctx context.Context, preparedConfig interface{}, dir string, exclude, include []*regexp.Regexp, variables map[string]interface{}) (interface{}, error) {
	switch t := preparedConfig.(type) {
	case string:
		return ResolveExpressions(ctx, t, dir, variables)
	case map[string]interface{}:
		err := walk.Walk(t, expressionMatchFn, func(path, value string) (interface{}, error) {
			if ExcludedPath(path, exclude, include) {
				return value, nil
			}

			return ResolveExpressions(ctx, value, dir, variables)
		})
		if err != nil {
			return nil, err
		}

		return t, nil
	case []interface{}:
		for i := range t {
			var err error
			t[i], err = ResolveAllExpressions(ctx, t[i], dir, exclude, include, variables)
			if err != nil {
				return nil, err
			}
		}

		return t, nil
	}

	return preparedConfig, nil
}

func ResolveExpressions(ctx context.Context, value, dir string, variables map[string]interface{}) (interface{}, error) {
	matches := ExpressionMatchRegex.FindAllStringSubmatch(value, -1)
	if len(matches) == 0 {
		return value, nil
	}

	vars := map[string]string{}
	for k, v := range variables {
		vars[k] = fmt.Sprintf("%v", v)
	}

	out := value
	for _, match := range matches {
		if len(match) != 2 {
			continue
		} else if value[1] == '$' {
			return value[1:], nil
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		envVars := []string{}
		envVars = append(envVars, DevSpaceSkipPreloadEnv+"=true")
		envVars = append(envVars, os.Environ()...)
		err := engine.ExecuteSimpleShellCommand(ctx, dir, env.NewVariableEnvProvider(expand.ListEnviron(envVars...), vars), stdout, stderr, nil, match[1], os.Args[1:]...)
		if err != nil {
			if len(strings.TrimSpace(stdout.String())) == 0 && len(strings.TrimSpace(stderr.String())) == 0 {
				if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
					out = "false"
				} else if status, ok := interp.IsExitStatus(err); ok && status == 1 {
					out = "false"
				} else {
					return nil, fmt.Errorf("error executing config expression %s: %v", match[1], err)
				}
			} else {
				return nil, fmt.Errorf("error executing config expression %s: %v (stdout: %s, stderr: %s)", match[1], err, stdout.String(), stderr.String())
			}
		} else {
			stdOut := stdout.String()
			if value[1] != '#' {
				stdOut = strings.TrimSpace(stdOut)
			}

			out = strings.Replace(out, match[0], stdOut, 1)
			if out == "" {
				out = "true"
			}
		}
	}

	// try to convert to an object
	if value[1] != '!' && value[2] != '!' {
		// is boolean?
		a, err := strconv.ParseBool(out)
		if err == nil {
			return a, nil
		}

		// is int?
		i, err := strconv.Atoi(out)
		if err == nil {
			return i, nil
		}

		// is null?
		if out == "" || out == "null" || out == "undefined" {
			return nil, nil
		}

		// is yaml object?
		m := map[string]interface{}{}
		err = yamlutil.Unmarshal([]byte(out), &m)
		if err == nil {
			return m, nil
		}

		// is yaml array?
		arr := []interface{}{}
		err = yamlutil.Unmarshal([]byte(out), &arr)
		if err == nil {
			return arr, nil
		}
	}

	return out, nil
}
