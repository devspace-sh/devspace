package examples

import (
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// RunQuickstart runs the test for the quickstart example
func RunDependencies(f *customFactory) error {
	log.GetInstance().Info("Run Dependencies")

	err := RunTest(f, "dependencies", nil)
	if err != nil {
		return err
	}

	return nil
}
