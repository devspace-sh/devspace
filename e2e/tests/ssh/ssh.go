package ssh

import (
	"context"
	"os"
	"os/exec"
	"time"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = DevSpaceDescribe("ssh", func() {
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

	ginkgo.It("devspace dev should start an SSH service", func(ctx context.Context) {
		tempDir, err := framework.CopyToTempDir("tests/ssh/testdata/ssh-simple")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("ssh")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(framework.ExpectDeleteNamespace, kubeClient, ns)

		// create a new dev command and start it
		done := make(chan error)
		cancelCtx, cancel := context.WithCancel(ctx)
		ginkgo.DeferCleanup(cancel)

		go func() {
			defer ginkgo.GinkgoRecover()

			devCmd := &cmd.RunPipelineCmd{
				GlobalFlags: &flags.GlobalFlags{
					NoWarn:    true,
					Namespace: ns,
				},
				Pipeline: "dev",
				Ctx:      cancelCtx,
			}

			done <- devCmd.RunDefault(f)
		}()

		// Wait for the dev session to start
		gomega.Eventually(func(g gomega.Gomega) {
			_, err := os.ReadFile("started")
			g.Expect(err).NotTo(gomega.HaveOccurred())
		}).
			WithPolling(time.Second).
			WithTimeout(time.Second * 60).
			Should(gomega.Succeed())

		// connect to the SSH server
		gomega.Eventually(func(g gomega.Gomega) {
			cmd := exec.Command("ssh", "test.ssh-simple.devspace", "ls")
			err := cmd.Run()
			g.Expect(err).NotTo(gomega.HaveOccurred())
		}).
			WithPolling(time.Second).
			WithTimeout(time.Second * 60).
			Should(gomega.Succeed())

		cancel()
		err = <-done
		framework.ExpectNoError(err)
	})

	ginkgo.It("devspace dev should NOT start an SSH service when disabled with a variable", func(ctx context.Context) {
		tempDir, err := framework.CopyToTempDir("tests/ssh/testdata/ssh-variable")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("ssh-without-service")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(framework.ExpectDeleteNamespace, kubeClient, ns)

		// create a new dev command and start it
		done := make(chan error)
		cancelCtx, cancel := context.WithCancel(ctx)
		ginkgo.DeferCleanup(cancel)

		go func() {
			defer ginkgo.GinkgoRecover()

			devCmd := &cmd.RunPipelineCmd{
				GlobalFlags: &flags.GlobalFlags{
					NoWarn:    true,
					Debug:     true,
					Namespace: ns,
					Vars:      []string{"SSH=false"},
				},
				Pipeline: "dev",
				Ctx:      cancelCtx,
			}

			done <- devCmd.RunDefault(f)
		}()

		// Wait for the dev session to start
		gomega.Eventually(func(g gomega.Gomega) {
			_, err := os.ReadFile("started")
			g.Expect(err).NotTo(gomega.HaveOccurred())
		}).
			WithPolling(time.Second).
			WithTimeout(time.Second * 60).
			Should(gomega.Succeed())

		gomega.Eventually(func(g gomega.Gomega) {
			cmd := exec.Command("ssh", "test.ssh-variable.devspace", "ls")
			out, err := cmd.CombinedOutput()
			output := string(out)
			g.Expect(err).To(gomega.HaveOccurred())
			g.Expect(output).To(
				gomega.Or(
					gomega.ContainSubstring("Could not resolve hostname test.ssh-variable.devspace"),
					gomega.ContainSubstring("ssh: connect to host localhost port 10023"),
				),
			)
		}).
			WithPolling(time.Second).
			WithTimeout(time.Second * 60).
			Should(gomega.Succeed())

		cancel()
		cmdErr := <-done
		framework.ExpectNoError(cmdErr)
	})

	ginkgo.It("devspace dev should start an SSH service when enabled with a variable", func(ctx context.Context) {
		tempDir, err := framework.CopyToTempDir("tests/ssh/testdata/ssh-variable")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("ssh-with-service")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(framework.ExpectDeleteNamespace, kubeClient, ns)

		// create a new dev command and start it
		done := make(chan error)
		cancelCtx, cancel := context.WithCancel(ctx)
		ginkgo.DeferCleanup(cancel)

		go func() {
			defer ginkgo.GinkgoRecover()

			devCmd := &cmd.RunPipelineCmd{
				GlobalFlags: &flags.GlobalFlags{
					NoWarn:    true,
					Debug:     true,
					Namespace: ns,
					Vars:      []string{"SSH=true"},
				},
				Pipeline: "dev",
				Ctx:      cancelCtx,
			}

			done <- devCmd.RunDefault(f)
		}()

		// Wait for the dev session to start
		gomega.Eventually(func(g gomega.Gomega) {
			_, err := os.ReadFile("started")
			g.Expect(err).NotTo(gomega.HaveOccurred())
		}).
			WithPolling(time.Second).
			WithTimeout(time.Second * 60).
			Should(gomega.Succeed())

		// connect to the SSH server
		gomega.Eventually(func(g gomega.Gomega) {
			cmd := exec.Command("ssh", "test.ssh-variable.devspace", "ls")
			err := cmd.Run()
			g.Expect(err).NotTo(gomega.HaveOccurred())
		}).
			WithPolling(time.Second).
			WithTimeout(time.Second * 60).
			Should(gomega.Succeed())

		cancel()
		cmdErr := <-done
		framework.ExpectNoError(cmdErr)
	})
})
