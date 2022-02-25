package walk

import (
	"fmt"
	"strconv"
)

// ReplaceFn defines the replace function
type ReplaceFn func(path, value string) (interface{}, error)

// MatchFn defines the match function
type MatchFn func(key, value string) bool

// Walk walks over an interface and replaces keys that match the match function with the replace function
func Walk(d map[string]interface{}, match MatchFn, replace ReplaceFn) error {
	return doWalk("", d, match, replace)
}

// WalkStringMap walks over an interface and replaces keys that match the match function with the replace function
func WalkStringMap(d map[string]interface{}, match MatchFn, replace ReplaceFn) error {
	return doWalk("", d, match, replace)
}

func doWalk(path string, d interface{}, match MatchFn, replace ReplaceFn) error {
	var err error

	switch t := d.(type) {
	case []interface{}:
		for idx, val := range t {
			newPath := path + "/" + strconv.Itoa(idx)
			value, ok := val.(string)
			if !ok {
				err = doWalk(newPath, val, match, replace)
				if err != nil {
					return err
				}

				continue
			}

			if match(fmt.Sprintf("[%d]", idx), value) {
				t[idx], err = replace(newPath, value)
				if err != nil {
					return err
				}
			}
		}
	case map[string]interface{}:
		for k, v := range t {
			key := k
			newPath := path + "/" + key
			value, ok := v.(string)
			if !ok {
				err = doWalk(newPath, v, match, replace)
				if err != nil {
					return err
				}

				continue
			}

			if match(key, value) {
				t[k], err = replace(newPath, value)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
