package util

// Filter removes matching strings from a string slice
func Filter(strings []string, predicate func(int, string) bool) (ret []string) {
	for i, s := range strings {
		if predicate(i, s) {
			ret = append(ret, s)
		}
	}
	return
}
