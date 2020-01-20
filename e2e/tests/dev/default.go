package dev

import (
	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

var devConfig cmd.DevCmd

// RunDefault runs the test for the default dev test
func RunDefault(f *customFactory, logger log.Logger) error {
	logger.Info("Run sub test 'default' of test 'dev'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	client, err := f.NewKubeClientFromContext("", f.Namespace, false)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	// The client is saved in the factory ONCE for each sub test
	f.Client = client

	err = beforeTest(f, logger, "tests/dev/testdata/default")
	defer afterTest(f)
	if err != nil {
		return errors.Errorf("sub test 'default' of 'dev' test failed: %s %v", f.GetLogContents(), err)
	}

	// serviceClient, err := setupServiceClient(f)
	// if err != nil {
	// 	return errors.Errorf("Unable to create fake service client: %v", err)
	// }

	devConfig = cmd.DevCmd{
		GlobalFlags: &flags.GlobalFlags{
			Namespace: f.Namespace,
		},
		UI:             false,
		Terminal:       true,
		Sync:           true,
		Portforwarding: true,
		ForceBuild:     true,
	}

	err = devConfig.Run(f, nil, nil)
	defer close(f.interrupt)
	if err != nil {
		return errors.Errorf("Error while running dev command: %v", err)
	}

	return nil
}
