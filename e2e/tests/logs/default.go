package logs

import (
	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/pkg/errors"
	"strings"
	"time"
)

func runDefault(f *customFactory) error {
	lc := &cmd.LogsCmd{
		GlobalFlags: &flags.GlobalFlags{
			Namespace: f.namespace,
		},
		LastAmountOfLines: 1,
	}

	done := utils.Capture()

	err := lc.RunLogs(nil, nil)
	if err != nil {
		return err
	}

	time.Sleep(time.Second * 5)

	capturedOutput, err := done()
	if err != nil {
		return err
	}

	if strings.Index(capturedOutput, "blabla world") == -1 {
		return errors.Errorf("capturedOutput '%v' is different than output 'blabla world' for the enter cmd", capturedOutput)
	}

	return nil
}
