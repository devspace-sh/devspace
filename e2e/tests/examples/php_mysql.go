package examples

import (
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/pkg/errors"
)

// RunPhpMysql runs the test for the quickstart example
func RunPhpMysql(f *customFactory, logger log.Logger) error {
	logger.Info("Run sub test 'php-mysql-example' of test 'examples'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	err := beforeTest(f, "../examples/php-mysql-example")
	defer afterTest(f)
	if err != nil {
		return errors.Errorf("sub test 'php-mysql-example' of 'examples' test failed: %s %v", f.GetLogContents(), err)
	}

	err = RunTest(f, nil)
	if err != nil {
		return errors.Errorf("sub test 'php-mysql-example' of 'examples' test failed: %s %v", f.GetLogContents(), err)
	}

	return nil
}
