package kubectl

// ReplaceFn defines the replace function
type ReplaceFn func(value string) string

// MatchFn defines the match function
type MatchFn func(key, value string) bool

// Walk walks over an interface and replaces keys that match the match function with the replace function
func Walk(d interface{}, match MatchFn, replace ReplaceFn) {
	switch t := d.(type) {
	case []interface{}:
		for _, val := range t {
			Walk(val, match, replace)
		}
	case map[interface{}]interface{}:
		for k, v := range t {
			key := k.(string)
			value, ok := v.(string)
			if ok == false {
				Walk(v, match, replace)
				continue
			}

			if match(key, value) {
				t[k] = replace(value)
			}
		}
	}
}
