package examples

import (
	"bytes"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// RunQuickstart runs the test for the quickstart example
func RunQuickstart(f *customFactory, logger log.Logger) error {
	buff := &bytes.Buffer{}
	f.cacheLogger = log.NewStreamLogger(buff, logrus.InfoLevel)

	logger.Info("Run sub test 'quickstart' of test 'examples'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	err := RunTest(f, "quickstart", nil)
	if err != nil {
		return errors.Errorf("sub test 'quickstart' of 'examples' test failed: %s %v", buff.String(), err)
	}

	return nil
}
