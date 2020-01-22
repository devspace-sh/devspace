package build

import (
	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

func runDefault(f *customFactory, logger log.Logger) error {
	logger.Info("Run sub test 'default' of test 'build'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	err := beforeTest(f, "default")
	defer afterTest(f)
	if err != nil {
		return errors.Errorf("test 'build' failed: %s %v", f.GetLogContents(), err)
	}

	bc := &cmd.BuildCmd{
		GlobalFlags: &flags.GlobalFlags{},
	}

	err = bc.Run(f, nil, nil)
	if err != nil {
		return err
	}

	imagesExpected := 1
	imagesCount := len(f.builtImages)
	if imagesCount != imagesExpected {
		return errors.Errorf("built images expected: %v, found: %v", imagesExpected, imagesCount)
	}

	f.builtImages = map[string]string{}

	return nil
}
