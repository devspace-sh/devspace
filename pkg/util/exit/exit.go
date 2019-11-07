package exit

import (
	"fmt"
)

// ReturnCodeError is used to return a non zero exit code error
type ReturnCodeError struct {
	ExitCode int
}

// Error implements interface
func (e *ReturnCodeError) Error() string {
	return fmt.Sprintf("exit code %d", e.ExitCode)
}
