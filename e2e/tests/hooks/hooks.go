package hooks

import (
	"os"
	"sync"
	"time"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/onsi/ginkgo"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = DevSpaceDescribe("hooks", func() {
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

	ginkgo.It("should execute hook once", func() {
		tempDir, err := framework.CopyToTempDir("tests/hooks/testdata/once")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("hooks")
		framework.ExpectNoError(err)
		defer func() {
			err := kubeClient.DeleteNamespace(ns)
			framework.ExpectNoError(err)
		}()

		// waitGroup for both commands
		waitGroup := sync.WaitGroup{}

		// create first dev command
		interrupt1, stop1 := framework.InterruptChan()
		defer stop1()
		devCmd1 := &cmd.DevCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Portforwarding: true,
			Sync:           true,
			Interrupt:      interrupt1,
		}

		// start the command
		waitGroup.Add(1)
		go func() {
			defer ginkgo.GinkgoRecover()
			defer waitGroup.Done()
			err = devCmd1.Run(f, nil)
			framework.ExpectNoError(err)
		}()

		// Read the 'once' hook output
		onceOutput1 := ""
		err = wait.PollImmediate(time.Second, time.Minute*2, func() (done bool, err error) {
			onceOutput1, err = kubeClient.ExecByImageSelector("node", ns, []string{"cat", "/app/once.out"})
			if err != nil {
				return false, nil
			}

			return onceOutput1 != "", nil
		})
		framework.ExpectNoError(err)

		// Read the 'each' hook output
		eachOutput1 := ""
		err = wait.PollImmediate(time.Second, time.Minute*2, func() (done bool, err error) {
			eachOutput1, err = kubeClient.ExecByImageSelector("node", ns, []string{"cat", "/app/each.out"})
			if err != nil {
				return false, nil
			}

			return eachOutput1 != "", nil
		})
		framework.ExpectNoError(err)

		// stop first command
		stop1()

		// create second dev command
		interrupt2, stop2 := framework.InterruptChan()
		defer stop2()
		devCmd2 := &cmd.DevCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Portforwarding: true,
			Sync:           true,
			Interrupt:      interrupt2,
		}

		// start the command
		waitGroup.Add(1)
		go func() {
			defer ginkgo.GinkgoRecover()
			defer waitGroup.Done()
			err = devCmd2.Run(f, nil)
			framework.ExpectNoError(err)
		}()

		// Wait for 'each' hook output to change
		eachOutput2 := ""
		err = wait.PollImmediate(time.Second, time.Minute*2, func() (done bool, err error) {
			eachOutput2, err = kubeClient.ExecByImageSelector("node", ns, []string{"cat", "/app/each.out"})
			if err != nil {
				return false, nil
			}

			return eachOutput2 != eachOutput1, nil
		})
		framework.ExpectNoError(err)

		// Read the 'once' hook output again
		onceOutput2 := ""
		err = wait.PollImmediate(time.Second, time.Minute*2, func() (done bool, err error) {
			onceOutput2, err = kubeClient.ExecByImageSelector("node", ns, []string{"cat", "/app/once.out"})
			if err != nil {
				return false, nil
			}

			return onceOutput2 != "", nil
		})
		framework.ExpectNoError(err)

		// stop second command
		stop2()

		// Verify that the 'once' hook did not run again
		framework.ExpectEqual(onceOutput1, onceOutput2)

		// wait for the command to finish
		waitGroup.Wait()
	})
})
