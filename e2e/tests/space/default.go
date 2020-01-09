package space

import (
	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/create"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/cmd/list"
	"github.com/devspace-cloud/devspace/cmd/remove"
	"github.com/devspace-cloud/devspace/cmd/use"
	"github.com/devspace-cloud/devspace/e2e/utils"
	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

func runDefault(f *customFactory, logger log.Logger) error {
	ls := &list.SpacesCmd{}
	cs := &create.SpaceCmd{}
	rs := &remove.SpaceCmd{}
	uc := &use.ContextCmd{}

	deployConfig := &cmd.DeployCmd{
		GlobalFlags: &flags.GlobalFlags{
			NoWarn:      true,
			KubeContext: "devspace-create-delete-space-testing-test",
		},
		SkipPush: true,
	}

	spaceName := "create-delete-space-testing-test"

	provider, err := cloudpkg.GetProvider(ls.Provider, logger)
	if err != nil {
		return err
	}
	spaces, err := provider.Client().GetSpaces()
	if err != nil {
		return err
	}

	spaceIsExist := utils.SpaceExists(spaceName, spaces)
	if spaceIsExist {
		err = rs.RunRemoveCloudDevSpace(f, nil, []string{spaceName})
		if err != nil {
			return err
		}
	}

	errcs := cs.RunCreateSpace(f, nil, []string{spaceName})
	defer clean(f, spaceName)
	erruc := uc.RunUseContext(f, nil, []string{f.previousContext})
	if errcs != nil {
		return err
	}
	if erruc != nil {
		return err
	}

	err = deployConfig.Run(f, nil, nil)
	if err != nil {
		return err
	}

	// Checking if pods are running correctly
	err = utils.AnalyzePods(f.client, f.client.Namespace(), f.cacheLogger)
	if err != nil {
		return err
	}

	return nil
}

func clean(f *customFactory, spaceName string) {
	rs := &remove.SpaceCmd{}

	err := rs.RunRemoveCloudDevSpace(f, nil, []string{spaceName})
	if err != nil {
		panic(err)
	}
}
