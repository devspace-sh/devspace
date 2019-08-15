package vars

import (
	"regexp"
	"strconv"
)

// VarMatchRegex is the regex to check if a value matches the devspace var format
var VarMatchRegex = regexp.MustCompile("(\\$+\\{[^\\}]+\\})")

// ReplaceVarFn defines the replace function
type ReplaceVarFn func(value string) (string, error)

// ParseString parses a given string, calls replace var on found variables and returns the replaced string
func ParseString(value string, replace ReplaceVarFn) (interface{}, error) {
	matches := VarMatchRegex.FindAllStringIndex(value, -1)

	// No vars found
	if len(matches) == 0 {
		return value, nil
	}

	newValue := value[:matches[0][0]]
	for index, match := range matches {
		var (
			matchStr    = value[match[0]:match[1]]
			newMatchStr string
			err         error
		)

		if matchStr[0] == '$' && matchStr[1] == '$' {
			newMatchStr = matchStr[1:]
		} else {
			newMatchStr, err = replace(matchStr[2 : len(matchStr)-1])
			if err != nil {
				return "", err
			}
		}

		newValue += newMatchStr
		if index+1 >= len(matches) {
			newValue += value[match[1]:]
		} else {
			newValue += value[match[1]:matches[index+1][0]]
		}
	}

	// Try to convert new value to boolean or integer
	if i, err := strconv.Atoi(newValue); err == nil {
		return i, nil
	} else if b, err := strconv.ParseBool(newValue); err == nil {
		return b, nil
	}

	return newValue, nil
}
