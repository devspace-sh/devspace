package variable

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/sirupsen/logrus"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/survey"
)

// NewUndefinedVariable creates a new variable that is loaded without definition
func NewUndefinedVariable(name string, localCache localcache.Cache, log log.Logger) Variable {
	return &undefinedVariable{
		name:       name,
		localCache: localCache,
		log:        log,
	}
}

type undefinedVariable struct {
	name       string
	localCache localcache.Cache
	log        log.Logger
}

func (u *undefinedVariable) Load(ctx context.Context, _ *latest.Variable) (interface{}, error) {
	// Is in environment?
	if os.Getenv(u.name) != "" {
		return convertStringValue(os.Getenv(u.name)), nil
	}

	// Is in generated config?
	if v, ok := u.localCache.GetVar(u.name); ok {
		return convertStringValue(v), nil
	}

	// is logger silent
	if u.log == log.Discard || u.log.GetLevel() < logrus.InfoLevel {
		return "", nil
	}

	// Ask for variable
	val, err := askQuestion(&latest.Variable{
		Question: "Please enter a value for " + u.name,
	}, u.log)
	if err != nil {
		return "", err
	}

	u.localCache.SetVar(u.name, val)
	return convertStringValue(val), nil
}

func convertStringValue(value string) interface{} {
	// Try to convert new value to boolean or integer
	if i, err := strconv.Atoi(value); err == nil {
		return i
	} else if b, err := strconv.ParseBool(value); err == nil {
		return b
	}

	return value
}

func askQuestion(variable *latest.Variable, log log.Logger) (string, error) {
	params := getParams(variable)

	answer, err := log.Question(params)
	if err != nil {
		return "", err
	}

	return answer, nil
}

func getParams(variable *latest.Variable) *survey.QuestionOptions {
	params := &survey.QuestionOptions{}

	if variable == nil {
		params.Question = "Please enter a value"
	} else {
		if variable.Question == "" {
			if variable.Name == "" {
				variable.Name = "variable"
			}

			params.Question = "Please enter a value for " + variable.Name
		} else {
			params.Question = variable.Question
		}

		if variable.Password {
			params.IsPassword = true
		}

		if variable.Default != "" {
			params.DefaultValue = fmt.Sprintf("%v", variable.Default)
		}
		if variable.Default != nil {
			params.DefaultValueSet = true
		}
		if len(variable.Options) > 0 {
			params.Options = variable.Options
			if variable.Default == nil {
				params.DefaultValue = params.Options[0]
			}
		} else if variable.ValidationPattern != "" {
			params.ValidationRegexPattern = variable.ValidationPattern

			if variable.ValidationMessage != "" {
				params.ValidationMessage = variable.ValidationMessage
			}
		}
	}
	return params
}
