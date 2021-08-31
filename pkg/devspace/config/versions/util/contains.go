package util

// Contains checks if a match is in the string slice, starting at the given index
func Contains(strings []string, predicate func(int, string) bool, idx int) bool {
	for i, s := range strings[idx:] {
		if predicate(i, s) {
			return true
		}
	}
	return false
}
