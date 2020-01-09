package deploy

import (
	"bytes"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

//Test 3 - kubectl
//1. deploy & kubectl (see quickstart-kubectl)
//2. purge (check if everything is deleted except namespace)

// RunKubectl runs the test for the kubectl test
func RunKubectl(f *customFactory, logger log.Logger) error {
	buff := &bytes.Buffer{}
	f.cacheLogger = log.NewStreamLogger(buff, logrus.InfoLevel)

	var buffString string
	buffString = buff.String()

	if f.Verbose {
		f.cacheLogger = logger
		buffString = ""
	}

	logger.Info("Run sub test 'kubectl' of test 'deploy'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	client, err := f.NewKubeClientFromContext("", f.Namespace, false)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	// The client is saved in the factory ONCE for each sub test
	f.Client = client

	ts := testSuite{
		test{
			name: "1. deploy & kubectl (see quickstart-kubectl)",
			deployConfig: &cmd.DeployCmd{
				GlobalFlags: &flags.GlobalFlags{
					Namespace: f.Namespace,
					NoWarn:    true,
				},
			},
			postCheck: nil,
		},
	}

	err = beforeTest(f, logger, "tests/deploy/testdata/kubectl")
	defer afterTest(f)
	if err != nil {
		return errors.Errorf("sub test 'kubectl' of 'deploy' test failed: %s %v", buffString, err)
	}

	for _, t := range ts {
		err := runTest(f, &t)
		utils.PrintTestResult("kubectl", t.name, err, logger)
		if err != nil {
			return errors.Errorf("sub test 'kubectl' of 'deploy' test failed: %s %v", buffString, err)
		}
	}

	err = testPurge(f)
	utils.PrintTestResult("kubectl", "purge", err, logger)
	if err != nil {
		return errors.Errorf("sub test 'kubectl' of 'deploy' test failed: %s %v", buffString, err)
	}

	return nil
}
