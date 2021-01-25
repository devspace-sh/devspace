package examples

import (
	"github.com/loft-sh/devspace/pkg/util/log"

	"github.com/pkg/errors"
)

// RunMicroservices runs the test for the kustomize example
func RunMicroservices(f *customFactory, logger log.Logger) error {
	logger.Info("Run sub test 'microservices' of test 'examples'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	err := beforeTest(f, "../examples/microservices")
	defer afterTest(f)
	if err != nil {
		return errors.Errorf("sub test 'microservices' of 'examples' test failed: %s %v", f.GetLogContents(), err)
	}

	err = RunTest(f, nil)
	if err != nil {
		return errors.Errorf("sub test 'microservices' of 'examples' test failed: %s %v", f.GetLogContents(), err)
	}

	return nil
}
