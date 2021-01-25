package examples

import (
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

// RunDependencies runs the test for the dependencies example
func RunDependencies(f *customFactory, logger log.Logger) error {
	logger.Info("Run sub test 'dependencies' of test 'examples'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	err := beforeTest(f, "../examples/dependencies")
	defer afterTest(f)
	if err != nil {
		return errors.Errorf("sub test 'dependencies' of 'examples' test failed: %s %v", f.GetLogContents(), err)
	}

	err = RunTest(f, nil)
	if err != nil {
		return errors.Errorf("sub test 'dependencies' of 'examples' test failed: %s %v", f.GetLogContents(), err)
	}

	return nil
}
