package walk

import (
	"fmt"
)

// ReplaceFn defines the replace function
type ReplaceFn func(value string) (interface{}, error)

// MatchFn defines the match function
type MatchFn func(key, value string) bool

// Walk walks over an interface and replaces keys that match the match function with the replace function
func Walk(d map[interface{}]interface{}, match MatchFn, replace ReplaceFn) error {
	return doWalk(d, match, replace)
}

// WalkStringMap walks over an interface and replaces keys that match the match function with the replace function
func WalkStringMap(d map[string]interface{}, match MatchFn, replace ReplaceFn) error {
	return doWalk(d, match, replace)
}

func doWalk(d interface{}, match MatchFn, replace ReplaceFn) error {
	var err error

	switch t := d.(type) {
	case []interface{}:
		for idx, val := range t {
			value, ok := val.(string)
			if ok == false {
				err = doWalk(val, match, replace)
				if err != nil {
					return err
				}

				continue
			}

			if match(fmt.Sprintf("[%d]", idx), value) {
				t[idx], err = replace(value)
				if err != nil {
					return err
				}
			}
		}
	case map[string]interface{}:
		for key, v := range t {
			value, ok := v.(string)
			if ok == false {
				err = doWalk(v, match, replace)
				if err != nil {
					return err
				}

				continue
			}

			if match(key, value) {
				t[key], err = replace(value)
				if err != nil {
					return err
				}
			}
		}
	case map[interface{}]interface{}:
		for k, v := range t {
			key := k.(string)
			value, ok := v.(string)
			if ok == false {
				err = doWalk(v, match, replace)
				if err != nil {
					return err
				}

				continue
			}

			if match(key, value) {
				t[k], err = replace(value)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
