package stdinutil

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

// GetFromStdinParams defines a question and its answerpatterns
type GetFromStdinParams struct {
	Question               string
	DefaultValue           string
	ValidationRegexPattern string
	Options                []string
	IsPassword             bool
}

// DefaultValidationRegexPattern is the default regex pattern to validate the input
var DefaultValidationRegexPattern = regexp.MustCompile("^.*$")

//GetFromStdin asks the user a question and returns the answer
func GetFromStdin(params *GetFromStdinParams) *string {
	var prompt survey.Prompt
	var result *string
	compiledRegex := DefaultValidationRegexPattern
	if params.ValidationRegexPattern != "" {
		compiledRegex = regexp.MustCompile(params.ValidationRegexPattern)
	}

	if params.Options != nil {
		prompt = &survey.Select{
			Message: params.Question,
			Options: params.Options,
			Default: params.DefaultValue,
		}
	} else if params.IsPassword {
		prompt = &survey.Password{
			Message: params.Question,
		}
	} else {
		prompt = &survey.Input{
			Message: params.Question,
			Default: params.DefaultValue,
		}
	}

	question := []*survey.Question{
		{
			Name:   "question",
			Prompt: prompt,
		},
	}

	if params.Options != nil {
		question[0].Validate = func(val interface{}) error {
			// since we are validating an Input, the assertion will always succeed
			if str, ok := val.(string); !ok || compiledRegex.MatchString(str) == false {
				return fmt.Errorf("Answer has to match pattern: %s", compiledRegex.String())
			}
			return nil
		}
	}

	for result == nil {
		// Ask it
		answers := struct {
			Question string
		}{}
		err := survey.Ask(question, &answers)
		if err != nil {
			if strings.HasPrefix(err.Error(), "Answer has to match pattern") {
				log.Info(err.Error())
				continue
			}

			// Keyboard interrupt
			os.Exit(0)
		}

		result = &answers.Question
	}

	return result
}
