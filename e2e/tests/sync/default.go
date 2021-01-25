package sync

import (
	"strings"
	"time"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/utils"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

func runDefault(f *customFactory, logger log.Logger) error {
	logger.Info("Run sub test 'default' of 'sync' test")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	sc := &cmd.SyncCmd{
		GlobalFlags: &flags.GlobalFlags{
			Namespace: f.Namespace,
			NoWarn:    true,
			Silent:    true,
		},
		LocalPath:             "./../foo",
		ContainerPath:         "/home",
		NoWatch:               true,
		DownloadOnInitialSync: true,
	}

	ec := &cmd.EnterCmd{
		GlobalFlags: &flags.GlobalFlags{
			Namespace: f.Namespace,
			NoWarn:    true,
			Silent:    true,
		},
		Container: "container-0",
	}

	err := sc.Run(f, nil, nil, nil)
	defer close(f.interrupt)
	if err != nil {
		return err
	}

	time.Sleep(time.Second * 5)

	done := utils.Capture()

	err = ec.Run(f, nil, nil, []string{"ls", "home"})
	if err != nil {
		return err
	}

	capturedOutput, err := done()
	if err != nil {
		return err
	}

	capturedOutput = strings.TrimSpace(capturedOutput)

	if strings.Index(capturedOutput, "bar.go") == -1 {
		return errors.Errorf("capturedOutput '%v' is different than output 'foo.go' for the sync cmd", capturedOutput)
	}

	return nil
}
