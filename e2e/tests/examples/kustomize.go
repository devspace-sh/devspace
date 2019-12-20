package examples

import (
	"bytes"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// RunKustomize runs the test for the kustomize example
func RunKustomize(f *customFactory, logger log.Logger) error {
	buff := &bytes.Buffer{}
	f.cacheLogger = log.NewStreamLogger(buff, logrus.InfoLevel)

	logger.Info("Run sub test 'kustomize' of test 'examples'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	err := RunTest(f, "kustomize", nil)
	if err != nil {
		return errors.Errorf("sub test 'kustomize' of 'examples' test failed: %s %v", buff.String(), err)
	}

	return nil
}
