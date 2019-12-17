package create_delete_space

import (
	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/create"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/cmd/remove"
	"github.com/devspace-cloud/devspace/cmd/use"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/pkg/errors"
)

func runDefault(f *customFactory) error {
	cs := &create.SpaceCmd{}
	rs := &remove.SpaceCmd{}
	uc := &use.ContextCmd{}

	deployConfig := &cmd.DeployCmd{
		GlobalFlags: &flags.GlobalFlags{
			NoWarn: true,
		},
		SkipPush: true,
	}

	err := cs.RunCreateSpace(nil, []string{"blabla-test-space"})

	err = deployConfig.Run(f, nil, nil)
	if err != nil {
		return err
	}

	client, err := f.NewKubeDefaultClient()
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	// Checking if pods are running correctly
	err = utils.AnalyzePods(client, client.Namespace())
	if err != nil {
		return err
	}

	err = rs.RunRemoveCloudDevSpace(nil, []string{"blabla-test-space"})
	if err != nil {
		return err
	}

	err = uc.RunUseContext(nil, []string{f.previousContext})
	if err != nil {
		return err
	}

	return nil
}
