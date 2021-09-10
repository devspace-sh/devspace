package expression

import (
	"bytes"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/kubectl/walk"
	"github.com/loft-sh/devspace/pkg/util/shell"
	"gopkg.in/yaml.v2"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ExpressionMatchRegex is the regex to check if a value matches the devspace var format
var ExpressionMatchRegex = regexp.MustCompile(`(?ms)^\$\#?\!?\((.+?)\)$`)

func expressionMatchFn(key, value string) bool {
	return ExpressionMatchRegex.MatchString(value)
}

func ResolveAllExpressions(preparedConfig map[interface{}]interface{}, dir string) error {
	err := walk.Walk(preparedConfig, expressionMatchFn, func(value string) (interface{}, error) {
		return ResolveExpressions(value, dir)
	})
	if err != nil {
		return err
	}

	return nil
}

func ResolveExpressions(value, dir string) (interface{}, error) {
	matches := ExpressionMatchRegex.FindAllStringSubmatch(value, -1)
	if len(matches) == 0 {
		return value, nil
	}

	out := value
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		err := shell.ExecuteShellCommand(match[1], os.Args[1:], dir, stdout, stderr, nil)
		if err != nil {
			return nil, fmt.Errorf("error executing config expression %s: %v (stdout: %s, stderr: %s)", match[1], err, stdout.String(), stderr.String())
		}

		stdOut := stdout.String()
		if value[1] != '#' {
			stdOut = strings.TrimSpace(stdOut)
		}

		out = strings.Replace(out, match[0], stdOut, 1)
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
		m := map[interface{}]interface{}{}
		err = yaml.Unmarshal([]byte(out), &m)
		if err == nil {
			return m, nil
		}

		// is yaml array?
		arr := []interface{}{}
		err = yaml.Unmarshal([]byte(out), &arr)
		if err == nil {
			return arr, nil
		}
	}

	return out, nil
}
