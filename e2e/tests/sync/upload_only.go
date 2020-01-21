package sync

import (
	"os"
	"time"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

func runUploadOnly(f *customFactory, logger log.Logger) error {
	logger.Info("Run sub test 'upload-only' of 'sync' test")
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
		UploadOnly:            true,
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

	err := sc.Run(f, nil, nil)
	defer close(f.interrupt)
	if err != nil {
		return err
	}

	time.Sleep(time.Second * 5)

	check0 := "bar.go"
	check1 := "alpha"
	check2 := "abc.txt"
	check3 := "/first"

	// Below checks if bar.go was uploaded to remote
	err = utils.IsFileOrFolderExistRemotely(f, ec, "home", check0)
	if err != nil {
		return err
	}

	// Create /alpha locally
	err = os.Mkdir(f.DirPath+"/foo/"+check1, os.ModePerm)
	if err != nil {
		return err
	}

	time.Sleep(time.Second * 5)

	// Check if /alpha was synced remotely
	err = utils.IsFileOrFolderExistRemotely(f, ec, "home", check1)
	if err != nil {
		return err
	}

	// Create /beta/abc.txt
	// Creates dir
	err = os.Mkdir(f.DirPath+"/foo/beta", os.ModePerm)
	if err != nil {
		return err
	}
	// Creates file
	_, err = os.OpenFile(f.DirPath+"/foo/beta/"+check2, os.O_RDONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}

	time.Sleep(time.Second * 5)

	// Check if /beta/abc.txt was synced remotely
	err = utils.IsFileOrFolderExistRemotely(f, ec, "home/beta", check2)
	if err != nil {
		return err
	}

	time.Sleep(time.Second * 5)

	// Check if /home/first was NOT created locally
	err = utils.IsFileOrFolderNotExist(f.DirPath + "/foo" + check3)
	if err != nil {
		return err
	}

	return nil
}
