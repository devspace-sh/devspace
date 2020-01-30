package provider

import (
	"github.com/devspace-cloud/devspace/cmd/add"
	"github.com/devspace-cloud/devspace/cmd/remove"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

func runDefault(f *customFactory, logger log.Logger) error {
	logger.Info("Run sub test 'default' of test 'add-provider'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	ap := &add.ProviderCmd{}
	rp := &remove.ProviderCmd{}

	err := ap.RunAddProvider(f, nil, []string{"test-provider.test"})
	if err != nil {
		return err
	}

	err = rp.RunRemoveCloudProvider(f, nil, []string{"test-provider.test"})
	if err != nil {
		return err
	}

	return nil
}
