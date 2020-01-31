package provider

import (
	"errors"

	"github.com/devspace-cloud/devspace/cmd/add"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

func runDefault(f *customFactory, logger log.Logger) error {
	logger.Info("Run sub test 'default' of test 'add-provider'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	ap := &add.ProviderCmd{}

	err := ap.RunAddProvider(f, nil, []string{"app.devspace.cloud"})
	if err != nil && err.Error() != "Provider app.devspace.cloud does already exist" {
		return err
	} else if err == nil {
		return errors.New("Expected error: 'Provider app.devspace.cloud does already exist', but found no error")
	}

	err = ap.RunAddProvider(f, nil, []string{"test-provider.test"})
	if err != nil {
		return err
	}

	return nil
}
