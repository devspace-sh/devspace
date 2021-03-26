package variable

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"os"
	"strconv"
)

// NewUndefinedVariable creates a new variable that is loaded without definition
func NewUndefinedVariable(name string, cache map[string]string, log log.Logger) Variable {
	return &undefinedVariable{
		name:  name,
		cache: cache,
		log:   log,
	}
}

type undefinedVariable struct {
	name  string
	cache map[string]string
	log   log.Logger
}

func (u *undefinedVariable) Load(definition *latest.Variable) (interface{}, error) {
	// Is in environment?
	if os.Getenv(u.name) != "" {
		return convertStringValue(os.Getenv(u.name)), nil
	}

	// Is in generated config?
	if _, ok := u.cache[u.name]; ok {
		return convertStringValue(u.cache[u.name]), nil
	}

	// Ask for variable
	var err error
	u.cache[u.name], err = askQuestion(&latest.Variable{
		Question: "Please enter a value for " + u.name,
	}, u.log)
	if err != nil {
		return "", err
	}

	return convertStringValue(u.cache[u.name]), nil
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

	answer, err := log.Question(params)
	if err != nil {
		return "", err
	}

	return answer, nil
}
