package create_delete_space

import (
	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/create"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/cmd/remove"
	"github.com/devspace-cloud/devspace/cmd/use"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

func runDefault(f *customFactory, logger log.Logger) error {
	cs := &create.SpaceCmd{}
	rs := &remove.SpaceCmd{}
	uc := &use.ContextCmd{}

	deployConfig := &cmd.DeployCmd{
		GlobalFlags: &flags.GlobalFlags{
			NoWarn: true,
		},
		SkipPush: true,
	}

	err := cs.RunCreateSpace(f, nil, []string{"blabla-test-space"})

	err = deployConfig.Run(f, nil, nil)
	if err != nil {
		return err
	}

	// Checking if pods are running correctly
	err = utils.AnalyzePods(f.client, f.client.Namespace(), f.cacheLogger)
	if err != nil {
		return err
	}

	err = rs.RunRemoveCloudDevSpace(f, nil, []string{"blabla-test-space"})
	if err != nil {
		return err
	}

	err = uc.RunUseContext(f, nil, []string{f.previousContext})
	if err != nil {
		return err
	}

	return nil
}
