package survey

import (
	"os"
	"regexp"

	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	surveypkg "gopkg.in/AlecAivazis/survey.v1"
)

// QuestionOptions defines a question and its options
type QuestionOptions struct {
	Question               string
	DefaultValue           string
	ValidationRegexPattern string
	ValidationMessage      string
	ValidationFunc         func(value string) error
	Options                []string
	IsPassword             bool
}

// DefaultValidationRegexPattern is the default regex pattern to validate the input
var DefaultValidationRegexPattern = regexp.MustCompile("^.*$")

var nextAnswers []string

// SetNextAnswer will set the next answer for the question function
// THIS SHOULD BE ONLY USED FOR UNIT TESTS
func SetNextAnswer(answer string) {
	nextAnswers = append(nextAnswers, answer)
}

// Question asks the user a question and returns the answer
func Question(params *QuestionOptions, log log.Logger) (string, error) {
	if log.GetLevel() < logrus.InfoLevel {
		return "", errors.Errorf("Cannot ask question '%s' because logger level is too low", params.Question)
	}

	var prompt surveypkg.Prompt
	compiledRegex := DefaultValidationRegexPattern
	if params.ValidationRegexPattern != "" {
		compiledRegex = regexp.MustCompile(params.ValidationRegexPattern)
	}

	if params.Options != nil {
		prompt = &surveypkg.Select{
			Message: params.Question + "\n",
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
			str, ok := val.(string)
			if !ok {
				return errors.New("Input was not a string")
			}

			// Check regex
			if compiledRegex.MatchString(str) == false {
				if params.ValidationMessage != "" {
					return errors.New(params.ValidationMessage)
				}

				return errors.Errorf("Answer has to match pattern: %s", compiledRegex.String())
			}

			// Check function
			if params.ValidationFunc != nil {
				err := params.ValidationFunc(str)
				if err != nil {
					if params.ValidationMessage != "" {
						return errors.New(params.ValidationMessage)
					}

					return errors.Errorf("%v", err)
				}
			}

			return nil
		}
	}

	// Ask it
	answers := struct {
		Question string
	}{}

	// Stop wait if there was any
	log.StopWait()

	if len(nextAnswers) != 0 {
		answer := nextAnswers[0]
		nextAnswers = nextAnswers[1:]
		return answer, nil
	}

	err := surveypkg.Ask(question, &answers)
	if err != nil {
		// Keyboard interrupt
		os.Exit(0)
	}

	return answers.Question, nil
}
