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

	var buffString string
	buffString = buff.String()

	if f.Verbose {
		f.cacheLogger = logger
		buffString = ""
	}

	logger.Info("Run sub test 'kustomize' of test 'examples'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	err := beforeTest(f, "../examples/kustomize")
	defer afterTest(f)
	if err != nil {
		return errors.Errorf("sub test 'kustomize' of 'examples' test failed: %s %v", buffString, err)
	}

	err = RunTest(f, nil)
	if err != nil {
		return errors.Errorf("sub test 'kustomize' of 'examples' test failed: %s %v", buffString, err)
	}

	return nil
}
