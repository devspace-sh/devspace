package survey

import "testing"

func TestSurvey(t *testing.T) {
	SetNextAnswer("testanswer")
	answer := Question(&QuestionOptions{
		Question: "Hello there",
	})
	if answer != "testanswer" {
		t.Fatalf("Expected testanswer, got %s", answer)
	}

	SetNextAnswer("testanswer")
	answer = Question(&QuestionOptions{
		Question:               "Hello there",
		Options:                []string{"test"},
		ValidationRegexPattern: "^test$",
	})
	if answer != "testanswer" {
		t.Fatalf("Expected testanswer, got %s", answer)
	}

	SetNextAnswer("testanswer")
	answer = Question(&QuestionOptions{
		IsPassword: true,
	})
	if answer != "testanswer" {
		t.Fatalf("Expected testanswer, got %s", answer)
	}
}
