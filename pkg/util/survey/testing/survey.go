package testing

import (
	"errors"

	surveypkg "github.com/devspace-cloud/devspace/pkg/util/survey"
)

// FakeSurvey is a fake survey that just returns predefined strings
type FakeSurvey struct {
	nextAnswers []string
}

// NewFakeSurvey creates a new fake survey
func NewFakeSurvey() *FakeSurvey {
	return &FakeSurvey{
		nextAnswers: []string{},
	}
}

// Question asks a question and returns a fake answer
func (f *FakeSurvey) Question(params *surveypkg.QuestionOptions) (string, error) {
	if len(f.nextAnswers) != 0 {
		answer := f.nextAnswers[0]
		f.nextAnswers = f.nextAnswers[1:]
		return answer, nil
	} else if params.DefaultValue != "" {
		return params.DefaultValue, nil
	}

	return "", errors.New("No answer to return specified")
}

// SetNextAnswer will set the next answer for the question function
func (f *FakeSurvey) SetNextAnswer(answer string) {
	f.nextAnswers = append(f.nextAnswers, answer)
}
