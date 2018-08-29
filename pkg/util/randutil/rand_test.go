package randutil

import (
	"regexp"
	"testing"
)

func TestGenerateRandomString(t *testing.T) {

	forbiddenCharsRegex := regexp.MustCompile("[^a-zA-Z0-9]")

	for i := 0; i < 10000; i++ {

		randString, err := GenerateRandomString(1)

		if err != nil {
			t.Error("Unexpected Error Occurred: " + err.Error())
			t.Fail()
		}

		t.Log(randString)

		if len(randString) != 1 {
			t.Error("Random String has unexpected length.\nExpected: 1\nActual: " + string(len(randString)))
			t.Fail()
		}

		if forbiddenCharsRegex.Match([]byte(randString)) {
			t.Error("Bad Character in generated String. Expected only letters and numbers. Acutal: " + randString)
			t.Fail()
		}
	}
}
