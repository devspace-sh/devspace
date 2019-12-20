package examples

import (
	"bytes"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// RunPhpMysql runs the test for the quickstart example
func RunPhpMysql(f *customFactory, logger log.Logger) error {
	buff := &bytes.Buffer{}
	f.cacheLogger = log.NewStreamLogger(buff, logrus.InfoLevel)

	logger.Info("Run sub test 'php-mysql-example' of test 'examples'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	err := RunTest(f, "php-mysql-example", nil)
	if err != nil {
		return errors.Errorf("sub test 'php-mysql-example' of 'examples' test failed: %s %v", buff.String(), err)
	}

	return nil
}
