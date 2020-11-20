package randutil

import (
	"crypto/rand"
	"encoding/base64"
	"regexp"
)

// GenerateRandomString returns a random strin containing only letters
func GenerateRandomString(length int) (string, error) {
	randBytes := make([]byte, length*2)

	_, randErr := rand.Read(randBytes)
	if randErr != nil {
		return "", randErr
	}

	regex := regexp.MustCompile("[^a-zA-Z0-9]")
	return regex.ReplaceAllString(base64.URLEncoding.EncodeToString(randBytes), "")[0:length], nil
}
