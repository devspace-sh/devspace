package examples

import (
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

// RunQuickstartKubectl runs the test for the quickstart example
func RunQuickstartKubectl(f *customFactory, logger log.Logger) error {
	logger.Info("Run sub test 'quickstart-kubectl' of test 'examples'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	err := beforeTest(f, "../examples/quickstart-kubectl")
	defer afterTest(f)
	if err != nil {
		return errors.Errorf("sub test 'quickstart-kubectl' of 'examples' test failed: %s %v", f.GetLogContents(), err)
	}

	err = RunTest(f, nil)
	if err != nil {
		return errors.Errorf("sub test 'quickstart-kubectl' of 'examples' test failed: %s %v", f.GetLogContents(), err)
	}

	return nil
}
