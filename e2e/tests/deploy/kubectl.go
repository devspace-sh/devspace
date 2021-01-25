package deploy

import (
	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/utils"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

//Test 3 - kubectl
//1. deploy & kubectl (see quickstart-kubectl)
//2. purge (check if everything is deleted except namespace)

// RunKubectl runs the test for the kubectl test
func RunKubectl(f *customFactory, logger log.Logger) error {
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
		return errors.Errorf("sub test 'kubectl' of 'deploy' test failed: %s %v", f.GetLogContents(), err)
	}

	for _, t := range ts {
		err := runTest(f, &t)
		utils.PrintTestResult("kubectl", t.name, err, logger)
		if err != nil {
			return errors.Errorf("sub test 'kubectl' of 'deploy' test failed: %s %v", f.GetLogContents(), err)
		}
	}

	err = testPurge(f)
	utils.PrintTestResult("kubectl", "purge", err, logger)
	if err != nil {
		return errors.Errorf("sub test 'kubectl' of 'deploy' test failed: %s %v", f.GetLogContents(), err)
	}

	return nil
}
