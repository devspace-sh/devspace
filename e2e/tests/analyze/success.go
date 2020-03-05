package analyze

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

func runSuccess(f *utils.BaseCustomFactory, logger log.Logger) error {
	logger.Info("Run sub test 'success' of test 'analyze'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	f.DirName = "success"

	ac := &cmd.AnalyzeCmd{
		GlobalFlags: &flags.GlobalFlags{
			Namespace: f.Namespace,
			NoWarn:    true,
		},
		Wait: true,
	}

	err := beforeTest(f)
	defer afterTest(f)
	if err != nil {
		return errors.Errorf("sub test 'success' of 'analyze' test failed: %s %v", f.GetLogContents(), err)
	}

	verboseHistory := f.Verbose
	f.Verbose = false

	err = ac.RunAnalyze(f, nil, nil)
	if err != nil {
		return errors.Errorf("err should be nil: %v", err)
	}

	expectedOutput := "No problems found"

	// If output does not contains all of expected output
	if strings.Index(f.GetLogContents(), expectedOutput) == -1 {
		return errors.Errorf("Expected output '%v' not found in following output: %s", expectedOutput, f.GetLogContents())
	}

	f.Verbose = verboseHistory

	return nil
}
