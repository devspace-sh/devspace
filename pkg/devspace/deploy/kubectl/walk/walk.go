package walk

import "fmt"

// ReplaceFn defines the replace function
type ReplaceFn func(path, value string) interface{}

// MatchFn defines the match function
type MatchFn func(path, key, value string) bool

// Walk walks over an interface and replaces keys that match the match function with the replace function
func Walk(d interface{}, match MatchFn, replace ReplaceFn) {
	doWalk(d, "", match, replace)
}

func doWalk(d interface{}, path string, match MatchFn, replace ReplaceFn) {
	switch t := d.(type) {
	case []interface{}:
		for idx, val := range t {
			doWalk(val, fmt.Sprintf("%s[%d]", path, idx), match, replace)
		}
	case map[interface{}]interface{}:
		for k, v := range t {
			key := k.(string)
			newPath := fmt.Sprintf("%s.%s", path, key)
			value, ok := v.(string)
			if ok == false {
				doWalk(v, newPath, match, replace)
				continue
			}

			if match(path, key, value) {
				t[k] = replace(newPath, value)
			}
		}
	}
}
