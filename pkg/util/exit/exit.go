package exit

import (
	"os"

	"github.com/devspace-cloud/devspace/pkg/util/analytics/cloudanalytics"
	"github.com/pkg/errors"
)

// Exit exits the runtime
func Exit(code int) {
	var err error
	if code != 0 {
		err = errors.Errorf("exit code %d", code)
	}

	cloudanalytics.SendCommandEvent(err)
	os.Exit(code)
}
