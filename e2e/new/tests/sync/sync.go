package sync

import (
	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/new/framework"
	"github.com/loft-sh/devspace/e2e/new/kube"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/randutil"
	"github.com/onsi/ginkgo"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/wait"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var _ = DevSpaceDescribe("sync", func() {
	initialDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// create a new factory
	var (
		f          factory.Factory
		kubeClient *kube.KubeHelper
	)

	ginkgo.BeforeEach(func() {
		f = framework.NewDefaultFactory()

		kubeClient, err = kube.NewKubeHelper()
		framework.ExpectNoError(err)
	})

	ginkgo.It("devspace sync should work with and without config", func() {
		// TODO:
		// test devspace sync command with devspace.yaml and without devspace.yaml
	})

	ginkgo.It("should execute a command after sync", func() {
		// TODO:
		// test config options dev.sync.onUpload.execRemote, dev.sync.onUpload.execRemote.onFileChange, dev.sync.onUpload.execRemote.onDirCreate, dev.sync.onUpload.execRemote.onBatch
		// test config options dev.sync.onDownload.execLocal, dev.sync.onDownload.execLocal.onFileChange, dev.sync.onDownload.execLocal.onDirCreate, dev.sync.onDownload.execLocal.onBatch
		// test config option dev.sync.onUpload.restartContainer
	})

	ginkgo.It("should sync to a pod and detect changes", func() {
		// TODO: test exclude / downloadExclude paths & file / folder deletion

		tempDir, err := framework.CopyToTempDir("tests/sync/testdata/dev-simple")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("sync")
		framework.ExpectNoError(err)
		defer kubeClient.DeleteNamespace(ns)

		// interrupt chan for the dev command
		interrupt, stop := framework.InterruptChan()
		defer stop()

		// create a new dev command
		devCmd := &cmd.DevCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Portforwarding: true,
			Sync:           true,
			Interrupt:      interrupt,
		}

		// start the command
		waitGroup := sync.WaitGroup{}
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			err = devCmd.Run(f, nil, nil, nil)
			framework.ExpectNoError(err)
		}()

		// wait until files were synced
		err = wait.PollImmediate(time.Second, time.Minute*2, func() (done bool, err error) {
			out, err := kubeClient.ExecByImageSelector("node", ns, []string{"cat", "/app/file1.txt"})
			if err != nil {
				return false, nil
			}

			return out == "Hello World", nil
		})
		framework.ExpectNoError(err)

		// check if sub file was synced
		out, err := kubeClient.ExecByImageSelector("node", ns, []string{"cat", "/app/folder1/file2.txt"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(out, "Hello World 2")

		// check if excluded file was synced
		out, err = kubeClient.ExecByImageSelector("node", ns, []string{"cat", "/app/test.txt"})
		framework.ExpectError(err)

		// write a file and check that it got synced
		payload := randutil.GenerateRandomString(10000)
		err = ioutil.WriteFile(filepath.Join(tempDir, "file3.txt"), []byte(payload), 0666)
		framework.ExpectNoError(err)

		// wait for sync
		err = wait.PollImmediate(time.Second, time.Minute*2, func() (done bool, err error) {
			out, err := kubeClient.ExecByImageSelector("node", ns, []string{"cat", "/app/file3.txt"})
			if err != nil {
				return false, nil
			}

			return out == payload, nil
		})
		framework.ExpectNoError(err)

		// check if file was downloaded through before hook
		_, err = ioutil.ReadFile(filepath.Join(tempDir, "file4.txt"))
		framework.ExpectError(err)
		framework.ExpectEqual(os.IsNotExist(err), true)

		// check if file was downloaded through after hook
		err = wait.PollImmediate(time.Second, time.Minute, func() (done bool, err error) {
			out, err := ioutil.ReadFile(filepath.Join(tempDir, "file5.txt"))
			if err != nil {
				if os.IsNotExist(err) == false {
					return false, err
				}

				return false, nil
			}

			return string(out) == "Hello World", nil
		})
		framework.ExpectNoError(err)

		// stop command
		stop()

		// wait for the command to finish
		waitGroup.Wait()
	})
})
