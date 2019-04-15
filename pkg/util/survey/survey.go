package survey

import (
	"errors"
	"fmt"
	"os"
	"regexp"

	surveypkg "gopkg.in/AlecAivazis/survey.v1"
)

// QuestionOptions defines a question and its options
type QuestionOptions struct {
	Question               string
	DefaultValue           string
	ValidationRegexPattern string
	ValidationMessage      string
	Options                []string
	IsPassword             bool
}

// DefaultValidationRegexPattern is the default regex pattern to validate the input
var DefaultValidationRegexPattern = regexp.MustCompile("^.*$")

// Question asks the user a question and returns the answer
func Question(params *QuestionOptions) string {
	var prompt surveypkg.Prompt
	compiledRegex := DefaultValidationRegexPattern
	if params.ValidationRegexPattern != "" {
		compiledRegex = regexp.MustCompile(params.ValidationRegexPattern)
	}

	if params.Options != nil {
		prompt = &surveypkg.Select{
			Message: params.Question,
			Options: params.Options,
			Default: params.DefaultValue,
		}
	} else if params.IsPassword {
		prompt = &surveypkg.Password{
			Message: params.Question,
		}
	} else {
		prompt = &surveypkg.Input{
			Message: params.Question,
			Default: params.DefaultValue,
		}
	}

	question := []*surveypkg.Question{
		{
			Name:   "question",
			Prompt: prompt,
		},
	}

	if params.Options == nil {
		question[0].Validate = func(val interface{}) error {
			// since we are validating an Input, the assertion will always succeed
			if str, ok := val.(string); !ok || compiledRegex.MatchString(str) == false {
				if params.ValidationMessage != "" {
					return errors.New(params.ValidationMessage)
				}

				return fmt.Errorf("Answer has to match pattern: %s", compiledRegex.String())
			}
			return nil
		}
	}

	// Ask it
	answers := struct {
		Question string
	}{}
	err := surveypkg.Ask(question, &answers)
	if err != nil {
		// Keyboard interrupt
		os.Exit(0)
	}

	return answers.Question
}
