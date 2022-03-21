package encoding

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

func SafeConcatName(name ...string) string {
	return SafeConcatNameMax(name, 63)
}

func SafeConcatGenerateName(name ...string) string {
	return SafeConcatNameMax(name, 53) + "-"
}

func SafeConcatNameMax(name []string, max int) string {
	fullPath := strings.Join(name, "-")
	if len(fullPath) > max {
		digest := sha256.Sum256([]byte(fullPath))
		return strings.Replace(fullPath[0:max-8]+"-"+hex.EncodeToString(digest[0:])[0:7], ".-", "-", -1)
	}
	return fullPath
}

var convertRegEx1 = regexp.MustCompile(`[\@/\.\:\s]+`)
var convertRegEx2 = regexp.MustCompile(`[^a-z0-9\-]+`)

func Convert(ID string) string {
	ID = strings.ToLower(ID)
	ID = convertRegEx1.ReplaceAllString(ID, "-")
	ID = convertRegEx2.ReplaceAllString(ID, "")
	return SafeConcatName(ID)
}

var UnsafeNameRegEx = regexp.MustCompile(`^[a-z0-9]+(?:-?[a-z0-9]+)*$`)

var UnsafeUpperNameRegEx = regexp.MustCompile(`^[A-Za-z0-9]+(?:[\-_]?[A-Za-z0-9])*$`)

func IsUnsafeUpperName(unsafeName string) bool {
	if UnsafeUpperNameRegEx.MatchString(unsafeName) {
		return false
	}
	return true
}

func IsUnsafeName(unsafeName string) bool {
	if UnsafeNameRegEx.MatchString(unsafeName) {
		return false
	}
	return true
}
