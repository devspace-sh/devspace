package vars

import (
	"fmt"
	"regexp"
)

// VarMatchRegex is the regex to check if a value matches the devspace var format
var VarMatchRegex = regexp.MustCompile(`(\$+!?\{[a-zA-Z0-9\-\_\.]+\})`)

// ReplaceVarFn defines the replace function
type ReplaceVarFn func(value string) (interface{}, error)

// ParseString parses a given string, calls replace var on found variables and returns the replaced string
func ParseString(value string, replace ReplaceVarFn) (interface{}, error) {
	if value == "" {
		return value, nil
	}

	matches := VarMatchRegex.FindAllStringIndex(value, -1)

	// No vars found
	if len(matches) == 0 {
		return value, nil
	}

	newValue := value[:matches[0][0]]
	forceString := false
	for index, match := range matches {
		var (
			matchStr    = value[match[0]:match[1]]
			newMatchStr string
		)

		if matchStr[0] == '$' && matchStr[1] == '$' {
			newMatchStr = matchStr[1:]
		} else {
			offset := 2
			if matchStr[1] == '!' {
				offset = 3
				forceString = true
			}

			replacedValue, err := replace(matchStr[offset : len(matchStr)-1])
			if err != nil {
				return "", err
			}

			switch v := replacedValue.(type) {
			case string:
				newMatchStr = v
			default:
				if forceString || len(matchStr) != len(value) {
					newMatchStr = fmt.Sprintf("%v", v)
				} else {
					return v, nil
				}
			}
		}

		newValue += newMatchStr
		if index+1 >= len(matches) {
			newValue += value[match[1]:]
		} else {
			newValue += value[match[1]:matches[index+1][0]]
		}
	}

	return newValue, nil
}
