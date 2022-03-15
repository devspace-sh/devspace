package sync

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/randutil"
	"github.com/onsi/ginkgo"
	"k8s.io/apimachinery/pkg/util/wait"
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

	ginkgo.It("devspace sync should override permissions on initial sync", func() {
		tempDir, err := framework.CopyToTempDir("tests/sync/testdata/permissions")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("sync")
		framework.ExpectNoError(err)
		defer func() {
			err := kubeClient.DeleteNamespace(ns)
			framework.ExpectNoError(err)
		}()

		// create a new dev command
		deployCmd := &cmd.DeployCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
		}

		// run the command
		err = deployCmd.Run(f)
		framework.ExpectNoError(err)

		// wait until busybox pod is reachable
		_, err = kubeClient.ExecByImageSelector("busybox", ns, []string{"sh", "-c", "mkdir /test_sync && echo -n 'echo \"Hello World!\"' > /test_sync/test.sh"})
		framework.ExpectNoError(err)

		// run single sync
		syncCmd := &cmd.SyncCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
				Debug:     true,
			},
			ContainerPath: "/test_sync",
			NoWatch:       true,
			ImageSelector: "busybox",
		}

		// run the command
		err = syncCmd.Run(f)
		framework.ExpectNoError(err)

		// check if script is executable
		_, err = kubeClient.ExecByImageSelector("busybox", ns, []string{"sh", "-c", "/test_sync/test.sh"})
		framework.ExpectError(err)

		// make script executable
		err = os.Chmod("test.sh", 0755)
		framework.ExpectNoError(err)

		// rerun sync command
		err = syncCmd.Run(f)
		framework.ExpectNoError(err)

		// make sure we got the right result this time
		out, err := kubeClient.ExecByImageSelector("busybox", ns, []string{"sh", "-c", "/test_sync/test.sh"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(string(out), "Hello World!\n")
	})

	ginkgo.It("devspace sync should work with and without config", func() {
		tempDir, err := framework.CopyToTempDir("tests/sync/testdata/no-config")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("sync")
		framework.ExpectNoError(err)
		defer func() {
			err := kubeClient.DeleteNamespace(ns)
			framework.ExpectNoError(err)
		}()

		// deploy app to sync
		deployCmd := &cmd.DeployCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:     true,
				Namespace:  ns,
				ConfigPath: "sync.yaml",
			},
		}
		err = deployCmd.Run(f)
		framework.ExpectNoError(err)

		// interrupt chan for the sync command
		interrupt, stop := framework.InterruptChan()
		defer stop()

		// sync with watch
		syncCmd := &cmd.SyncCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			ImageSelector: "node:13.14-alpine",
			ContainerPath: "/app",
			UploadOnly:    true,
			Polling:       true,
			Wait:          true,
			Interrupt:     interrupt,
		}

		// start the command
		waitGroup := sync.WaitGroup{}
		waitGroup.Add(1)
		go func() {
			defer ginkgo.GinkgoRecover()
			defer waitGroup.Done()

			err := syncCmd.Run(f)
			framework.ExpectNoError(err)
		}()

		// wait until files were synced
		framework.ExpectRemoteFileContents("node:13.14-alpine", ns, "/app/file1.txt", "Hello World\n")

		// stop sync
		stop()

		// wait for the command to finish
		waitGroup.Wait()
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
		defer func() {
			err := kubeClient.DeleteNamespace(ns)
			framework.ExpectNoError(err)
		}()

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
			defer ginkgo.GinkgoRecover()
			defer waitGroup.Done()
			err = devCmd.Run(f, nil)
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
		_, err = kubeClient.ExecByImageSelector("node", ns, []string{"cat", "/app/test.txt"})
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
				if !os.IsNotExist(err) {
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

	ginkgo.It("should sync to a pod and watch changes", func() {
		tempDir, err := framework.CopyToTempDir("tests/sync/testdata/sync-simple")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("sync")
		framework.ExpectNoError(err)
		defer func() {
			err := kubeClient.DeleteNamespace(ns)
			framework.ExpectNoError(err)
		}()

		// deploy app to sync
		deployCmd := &cmd.DeployCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:     true,
				Namespace:  ns,
				ConfigPath: "watch.yaml",
			},
		}
		err = deployCmd.Run(f)
		framework.ExpectNoError(err)

		// interrupt chan for the sync command
		interrupt, stop := framework.InterruptChan()
		defer stop()

		// sync with watch
		syncCmd := &cmd.SyncCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:     true,
				Namespace:  ns,
				ConfigPath: "watch.yaml",
			},
			Interrupt: interrupt,
		}

		// start the command
		waitGroup := sync.WaitGroup{}
		waitGroup.Add(1)
		go func() {
			defer ginkgo.GinkgoRecover()
			defer waitGroup.Done()

			err := syncCmd.Run(f)
			framework.ExpectNoError(err)
		}()

		// wait until files were synced
		framework.ExpectRemoteFileContents("node", ns, "/watch/file1.txt", "Hello World")

		// check if file was downloaded through after hook
		framework.ExpectLocalFileContents(filepath.Join(tempDir, "initial-sync-done.txt"), "Hello World")

		// write a file and check that it got synced
		payload := randutil.GenerateRandomString(10000)
		err = ioutil.WriteFile(filepath.Join(tempDir, "watching.txt"), []byte(payload), 0666)
		framework.ExpectNoError(err)
		framework.ExpectRemoteFileContents("node", ns, "/watch/watching.txt", payload)

		// stop command
		stop()

		// wait for the command to finish
		waitGroup.Wait()
	})

	ginkgo.It("should sync to a pod and not watch changes with --no-watch", func() {
		tempDir, err := framework.CopyToTempDir("tests/sync/testdata/sync-simple")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("sync")
		framework.ExpectNoError(err)
		defer func() {
			err := kubeClient.DeleteNamespace(ns)
			framework.ExpectNoError(err)
		}()

		// deploy app to sync
		deployCmd := &cmd.DeployCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:     true,
				Namespace:  ns,
				ConfigPath: "no-watch.yaml",
			},
		}
		err = deployCmd.Run(f)
		framework.ExpectNoError(err)

		// sync with no-watch
		syncCmd := &cmd.SyncCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:     true,
				Namespace:  ns,
				ConfigPath: "no-watch.yaml",
			},
			NoWatch:               true,
			DownloadOnInitialSync: true,
		}

		// start the command
		err = syncCmd.Run(f)
		framework.ExpectNoError(err)

		// wait until files were synced
		framework.ExpectRemoteFileContents("node", ns, "/no-watch/file1.txt", "Hello World")

		// check if file was downloaded correctly
		framework.ExpectLocalFileContents(filepath.Join(tempDir, "initial-sync-done-before.txt"), "Hello World")

		// check if file was downloaded through after hook
		framework.ExpectLocalFileNotFound(filepath.Join(tempDir, "initial-sync-done-after.txt"))
	})

	ginkgo.It("should sync to a pod container with --container and --container-path", func() {
		tempDir, err := framework.CopyToTempDir("tests/sync/testdata/sync-containers")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("sync")
		framework.ExpectNoError(err)
		defer func() {
			err := kubeClient.DeleteNamespace(ns)
			framework.ExpectNoError(err)
		}()

		// deploy app to sync
		deployCmd := &cmd.DeployCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:     true,
				Namespace:  ns,
				ConfigPath: "devspace.yaml",
			},
		}
		err = deployCmd.Run(f)
		framework.ExpectNoError(err)

		// sync with --container and --container-path
		interrupt, stop := framework.InterruptChan()
		defer stop()

		syncCmd := &cmd.SyncCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:     true,
				Namespace:  ns,
				ConfigPath: "devspace.yaml",
			},
			Container:     "container2",
			ContainerPath: "/app2",
			Interrupt:     interrupt,
		}

		// start the command
		waitGroup := sync.WaitGroup{}
		waitGroup.Add(1)
		go func() {
			defer ginkgo.GinkgoRecover()
			defer waitGroup.Done()
			err = syncCmd.Run(f)
			framework.ExpectNoError(err)
		}()

		// wait until files were synced
		framework.ExpectRemoteContainerFileContents("e2e=sync-containers", "container2", ns, "/app2/file1.txt", "Hello World")

		// write a file and check that it got synced
		payload := randutil.GenerateRandomString(10000)
		err = ioutil.WriteFile(filepath.Join(tempDir, "watching.txt"), []byte(payload), 0666)
		framework.ExpectNoError(err)
		framework.ExpectRemoteContainerFileContents("e2e=sync-containers", "container2", ns, "/app2/watching.txt", payload)

		// stop command
		stop()

		// wait for the command to finish
		waitGroup.Wait()
	})

	ginkgo.It("should sync to a pod container with excludeFile, downloadExcludeFile, and uploadExcludeFile configuration", func() {
		tempDir, err := framework.CopyToTempDir("tests/sync/testdata/sync-exclude-file")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("sync")
		framework.ExpectNoError(err)
		defer func() {
			err := kubeClient.DeleteNamespace(ns)
			framework.ExpectNoError(err)
		}()

		// deploy app to sync
		deployCmd := &cmd.DeployCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:     true,
				Namespace:  ns,
				ConfigPath: "devspace.yaml",
			},
		}
		err = deployCmd.Run(f)
		framework.ExpectNoError(err)

		interrupt, stop := framework.InterruptChan()
		defer stop()

		// sync command
		syncCmd := &cmd.SyncCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:     true,
				Namespace:  ns,
				ConfigPath: "devspace.yaml",
			},
			Interrupt: interrupt,
		}

		// start the command
		waitGroup := sync.WaitGroup{}
		waitGroup.Add(1)
		go func() {
			defer ginkgo.GinkgoRecover()
			defer waitGroup.Done()
			err = syncCmd.Run(f)
			framework.ExpectNoError(err)
		}()

		// wait for initial sync to complete
		framework.ExpectLocalFileContents(filepath.Join(tempDir, "initial-sync-done.txt"), "Hello World")

		// check that included file was synced
		framework.ExpectRemoteFileContents("node", ns, "/app/file-include.txt", "Hello World")

		// check that excluded file was not synced
		framework.ExpectRemoteFileNotFound("node", ns, "/app/file-exclude.txt")

		// check that upload exluded file was not synced
		framework.ExpectLocalFileContents(filepath.Join(tempDir, "file-upload-exclude.txt"), "Hello World")
		framework.ExpectRemoteFileNotFound("node", ns, "/app/file-upload-exclude.txt")

		// check that download excluded file was not synced
		framework.ExpectLocalFileNotFound(filepath.Join(tempDir, "file-download-exclude.txt"))
		framework.ExpectRemoteFileContents("node", ns, "/app/file-download-exclude.txt", "Hello World")

		// write a file and check that it got synced
		payload := randutil.GenerateRandomString(10000)
		err = ioutil.WriteFile(filepath.Join(tempDir, "watching.txt"), []byte(payload), 0666)
		framework.ExpectNoError(err)
		framework.ExpectRemoteFileContents("node", ns, "/app/watching.txt", payload)

		// stop command
		stop()

		// wait for the command to finish
		waitGroup.Wait()
	})
})
