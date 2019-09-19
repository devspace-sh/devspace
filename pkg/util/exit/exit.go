package exit

import (
	"fmt"
	"os"

	"github.com/devspace-cloud/devspace/pkg/util/analytics/cloudanalytics"
	"github.com/pkg/errors"
)

// ReturnCodeError is used to return a non zero exit code error
type ReturnCodeError struct {
	ExitCode int
}

// Error implements interface
func (e *ReturnCodeError) Error() string {
	return fmt.Sprintf("exit code %d", e.ExitCode)
}

// Exit exits the runtime
func Exit(code int) {
	var err error
	if code != 0 {
		err = errors.Errorf("exit code %d", code)
	}

	cloudanalytics.SendCommandEvent(err)
	os.Exit(code)
}
