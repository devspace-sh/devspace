package walk

import (
	"fmt"
)

// ReplaceFn defines the replace function
type ReplaceFn func(path, value string) (interface{}, error)

// MatchFn defines the match function
type MatchFn func(path, key, value string) bool

// Walk walks over an interface and replaces keys that match the match function with the replace function
func Walk(d map[interface{}]interface{}, match MatchFn, replace ReplaceFn) error {
	return doWalk(d, "", match, replace)
}

func doWalk(d interface{}, path string, match MatchFn, replace ReplaceFn) error {
	var err error

	switch t := d.(type) {
	case []interface{}:
		for idx, val := range t {
			newPath := fmt.Sprintf("%s[%d]", path, idx)
			value, ok := val.(string)
			if ok == false {
				err = doWalk(val, newPath, match, replace)
				if err != nil {
					return err
				}

				continue
			}

			if match(path, fmt.Sprintf("[%d]", idx), value) {
				t[idx], err = replace(newPath, value)
				if err != nil {
					return err
				}
			}
		}
	case map[interface{}]interface{}:
		for k, v := range t {
			key := k.(string)
			newPath := fmt.Sprintf("%s.%s", path, key)
			value, ok := v.(string)
			if ok == false {
				err = doWalk(v, newPath, match, replace)
				if err != nil {
					return err
				}

				continue
			}

			if match(path, key, value) {
				t[k], err = replace(newPath, value)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
