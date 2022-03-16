package merge

// Values is the type to go
type Values map[string]interface{}

// MergeInto takes the properties in src and merges them into Values. Maps
// are merged while values and arrays are replaced.
func (v Values) MergeInto(src Values) {
	for key, srcVal := range src {
		destVal, found := v[key]

		if found && istable(srcVal) && istable(destVal) {
			srcMap := srcVal.(map[string]interface{})
			destMap := destVal.(map[string]interface{})
			Values(destMap).MergeInto(Values(srcMap))
		} else {
			v[key] = srcVal
		}
	}
}

// istable is a special-purpose function to see if the present thing matches the definition of a YAML table.
func istable(v interface{}) bool {
	_, ok := v.(map[string]interface{})
	return ok
}
