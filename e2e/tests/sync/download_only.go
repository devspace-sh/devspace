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

func runDownloadOnly(f *customFactory, logger log.Logger) error {
	logger.Info("Run sub test 'download-only' of 'sync' test")
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
		DownloadOnly:          true,
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

	check0 := "bar.go"
	check1 := "/first/abc.txt"
	check2 := "/second"
	check3 := "/second/abc.txt"

	// Below checks if bar.go was NOT uploaded to remote
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

	if strings.Index(capturedOutput, check0) != -1 {
		return errors.Errorf("file '%s' should not have been uploaded to remote", check0)
	}

	// Check if /alpha/abc.txt was created locally
	err = utils.IsFileOrFolderExist(f.DirPath + "/foo" + check1)
	if err != nil {
		return err
	}

	err = ec.Run(f, nil, nil, []string{"mkdir", "home/second"})
	if err != nil {
		return err
	}

	time.Sleep(time.Second * 5)

	// Check if /second was created locally
	err = utils.IsFileOrFolderExist(f.DirPath + "/foo" + check2)
	if err != nil {
		return err
	}

	err = ec.Run(f, nil, nil, []string{"touch", "home/second/abc.txt"})
	if err != nil {
		return err
	}

	time.Sleep(time.Second * 5)

	// Check if /second/abc.txt was created locally
	err = utils.IsFileOrFolderExist(f.DirPath + "/foo" + check3)
	if err != nil {
		return err
	}

	return nil
}
