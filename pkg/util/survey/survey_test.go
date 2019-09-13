package survey

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	"gotest.tools/assert"
)

type testCase struct {
	name            string
	questions       []*QuestionOptions
	answersSet      []string
	expectedAnswers []string
}

func TestSurvey(t *testing.T) {
	testCases := []testCase{
		{
			name: "Two questions",
			questions: []*QuestionOptions{
				&QuestionOptions{
					Question: "Hello",
				},
				&QuestionOptions{
					Question:               "Hello",
					Options:                []string{"test"},
					ValidationRegexPattern: "^test$",
				},
			},
			answersSet:      []string{"World", "Universe"},
			expectedAnswers: []string{"World", "Universe"},
		},
		{
			name: "Password question",
			questions: []*QuestionOptions{
				&QuestionOptions{
					Question:   "Password please",
					IsPassword: true,
				},
			},
			answersSet:      []string{"Unsafe password"},
			expectedAnswers: []string{"Unsafe password"},
		},
	}

	for _, test := range testCases {
		nextAnswers = []string{}
		for _, answer := range test.answersSet {
			SetNextAnswer(answer)
		}

		for index, question := range test.questions {
			answer, _ := Question(question, log.GetInstance())
			assert.Equal(t, test.expectedAnswers[index], answer, "Wrong answer in testcase %s", test.name)
		}
	}
}
