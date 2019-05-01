package ptr

// String returns a pointer to a string variable
func String(val string) *string {
	return &val
}

// ReverseString returns a string from a string pointer
func ReverseString(val *string) string {
	if val == nil {
		return ""
	}

	return *val
}

// Int returns a pointer to an int variable
func Int(val int) *int {
	return &val
}

// Int32 returns a pointer to an int32 variable
func Int32(val int32) *int32 {
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

// ReverseBool returns a bool from a bool pointer
func ReverseBool(val *bool) bool {
	if val == nil {
		return false
	}

	return *val
}
