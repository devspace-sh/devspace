package sync

import (
	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/new/framework"
	"github.com/loft-sh/devspace/e2e/new/kube"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/onsi/ginkgo"
	"k8s.io/apimachinery/pkg/util/wait"
	"os"
	"sync"
	"time"
)

var _ = DevSpaceDescribe("devspace dev", func() {
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

	ginkgo.It("should sync to a pod and detect changes", func() {
		tempDir, err := framework.CopyToTempDir("tests/sync/testdata/dev-simple")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("sync")
		framework.ExpectNoError(err)
		// defer kubeClient.DeleteNamespace(ns)

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
		err = wait.Poll(time.Second, time.Minute*2, func() (done bool, err error) {
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

		// stop command
		stop()

		// wait for the command to finish
		waitGroup.Wait()
	})
})
