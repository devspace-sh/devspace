//go:build !darwin && !windows
// +build !darwin,!windows

package idle

// NewIdleGetter returns a new idle getter for windows
func NewIdleGetter() (Getter, error) {
	return nil, &unsupportedError{}
}
