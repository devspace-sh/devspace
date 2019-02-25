package ptr

// String returns a pointer to a string variable
func String(val string) *string {
	return &val
}

// Bool returns a pointer to a bool variable
func Bool(val bool) *bool {
	return &val
}
