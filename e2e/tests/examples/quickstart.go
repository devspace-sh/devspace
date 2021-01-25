package examples

import (
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

// RunQuickstart runs the test for the quickstart example
func RunQuickstart(f *customFactory, logger log.Logger) error {
	logger.Info("Run sub test 'quickstart' of test 'examples'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	err := beforeTest(f, "../examples/quickstart")
	defer afterTest(f)
	if err != nil {
		return errors.Errorf("sub test 'quickstart' of 'examples' test failed: %s %v", f.GetLogContents(), err)
	}

	err = RunTest(f, nil)
	if err != nil {
		return errors.Errorf("sub test 'quickstart' of 'examples' test failed: %s %v", f.GetLogContents(), err)
	}

	return nil
}
