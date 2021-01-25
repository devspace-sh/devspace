package deploy

import (
	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/utils"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

//Test 4 - helm
//1. deploy & helm (see quickstart) (v1beta5 no tiller)
//2. purge (check if everything is deleted except namespace)

// RunHelm runs the test for the kubectl test
func RunHelm(f *customFactory, logger log.Logger) error {
	logger.Info("Run sub test 'helm' of test 'deploy'")
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
			name: "1. deploy & helm (see quickstart) (v1beta5 no tiller)",
			deployConfig: &cmd.DeployCmd{
				GlobalFlags: &flags.GlobalFlags{
					Namespace: f.Namespace,
					NoWarn:    true,
				},
			},
			postCheck: nil,
		},
	}

	err = beforeTest(f, logger, "tests/deploy/testdata/helm")
	defer afterTest(f)
	if err != nil {
		return errors.Errorf("sub test 'helm' of 'deploy' test failed: %s %v", f.GetLogContents(), err)
	}

	for _, t := range ts {
		err := runTest(f, &t)
		utils.PrintTestResult("helm", t.name, err, logger)
		if err != nil {
			return errors.Errorf("sub test 'helm' of 'deploy' test failed: %s %v", f.GetLogContents(), err)
		}
	}

	err = testPurge(f)
	utils.PrintTestResult("helm", "purge", err, logger)
	if err != nil {
		return err
	}

	return nil
}
