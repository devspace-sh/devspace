package randutil

import (
	"regexp"
	"strconv"
	"testing"
)

func TestGenerateRandomString(t *testing.T) {
	forbiddenCharsRegex := regexp.MustCompile("[^a-zA-Z0-9]")
	for i := 0; i < 1000; i++ {
		randString := GenerateRandomString(1)
		if len(randString) != 1 {
			t.Error("Random String has unexpected length.\nExpected: 1\nActual: " + strconv.Itoa(len(randString)))
			t.Fail()
		}

		if forbiddenCharsRegex.Match([]byte(randString)) {
			t.Error("Bad Character in generated String. Expected only letters and numbers. Acutal: " + randString)
			t.Fail()
		}
	}
}
