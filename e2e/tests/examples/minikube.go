package examples

import (
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/pkg/errors"
)

// RunMinikube runs the test for the kustomize example
func RunMinikube(f *customFactory, logger log.Logger) error {
	logger.Info("Run sub test 'minikube' of test 'examples'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	err := beforeTest(f, "../examples/minikube")
	defer afterTest(f)
	if err != nil {
		return errors.Errorf("sub test 'minikube' of 'examples' test failed: %s %v", f.GetLogContents(), err)
	}

	err = RunTest(f, nil)
	if err != nil {
		return errors.Errorf("sub test 'minikube' of 'examples' test failed: %s %v", f.GetLogContents(), err)
	}

	return nil
}
