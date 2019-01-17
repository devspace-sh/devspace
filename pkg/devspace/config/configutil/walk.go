package configutil

// ReplaceFn defines the replace function
type ReplaceFn func(map[interface{}]interface{}) interface{}

// MatchFn defines the match function
type MatchFn func(map[interface{}]interface{}) bool

// Walk walks over an interface and replaces keys that match the match function with the replace function
func Walk(d interface{}, match MatchFn, replace ReplaceFn) interface{} {
	switch t := d.(type) {
	case []interface{}:
		for _, val := range t {
			Walk(val, match, replace)
		}
	case map[interface{}]interface{}:
		if match(t) {
			return replace(t)
		}

		for k, v := range t {
			replaced := Walk(v, match, replace)
			if replaced != nil {
				t[k] = replaced
			}
		}
	}

	return nil
}
