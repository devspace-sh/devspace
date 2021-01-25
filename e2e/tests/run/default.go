package run

import (
	"fmt"
	"strings"
	"time"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/utils"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

func runDefault(f *customFactory, logger log.Logger) error {
	logger.Info("Run test 'default' of 'run'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	sc := &cmd.SyncCmd{
		GlobalFlags: &flags.GlobalFlags{
			Namespace: f.Namespace,
			NoWarn:    true,
			Silent:    true,
		},
		LocalPath:     "./../foo",
		ContainerPath: "/home",
		NoWatch:       true,
	}

	rc := &cmd.RunCmd{
		GlobalFlags: &flags.GlobalFlags{
			Namespace: f.Namespace,
		},
	}

	err := sc.Run(f, nil, nil, nil)
	defer close(f.interrupt)
	if err != nil {
		return errors.Errorf("Error while running sync command: %s", err.Error())
	}

	ns := fmt.Sprintf("--namespace=%s", f.Namespace)
	time.Sleep(time.Second * 5)

	done := utils.Capture()

	err = rc.RunRun(f, nil, []string{"command-test", ns})
	if err != nil {
		return errors.Errorf("Error while running run command: %s", err.Error())
	}

	capturedOutput, err := done()
	if err != nil {
		return err
	}

	capturedOutput = strings.TrimSpace(capturedOutput)

	if strings.Index(capturedOutput, "bar.go") == -1 {
		return errors.Errorf("capturedOutput '%v' is different than output 'foo.go' for the run cmd", capturedOutput)
	}

	return nil
}
