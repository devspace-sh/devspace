package ptr

// String returns a pointer to a string variable
func String(val string) *string {
	return &val
}

// Int64 returns a pointer to an int64 variable
func Int64(val int64) *int64 {
	return &val
}

// Bool returns a pointer to a bool variable
func Bool(val bool) *bool {
	return &val
}
