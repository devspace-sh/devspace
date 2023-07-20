package proxycommands

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/onsi/ginkgo/v2"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/util/factory"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = DevSpaceDescribe("proxyCommands", func() {
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

	ginkgo.It("devspace dev should proxy commands to host machine", func() {
		tempDir, err := framework.CopyToTempDir("tests/proxycommands/testdata/proxycommands-simple")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("proxycommands")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		// create a new dev command and start it
		done := make(chan error)
		cancelCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

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

		// Check that the command is proxied to the host.
		var stdout, stderr bytes.Buffer
		cmd := exec.Command("uname", "-n")
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err = cmd.Run()
		framework.ExpectNoError(err)

		// Get the expected Pod hostname
		var pods *corev1.PodList
		err = wait.Poll(time.Second, time.Minute, func() (done bool, err error) {
			pods, err = kubeClient.RawClient().CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: "app.kubernetes.io/component=test"})
			if err != nil {
				return false, err
			}
			return len(pods.Items) > 0, nil
		})
		framework.ExpectNoError(err)
		podName := pods.Items[0].Name

		framework.ExpectLocalFileContents("host.out", stdout.String())
		framework.ExpectRemoteFileContents("alpine", ns, "container.out", fmt.Sprintf("%s\n", podName))

		cancel()

		err = <-done
		framework.ExpectNoError(err)
	})

	ginkgo.It("devspace dev should proxy commands to host machine without /usr/local/bin", func() {
		tempDir, err := framework.CopyToTempDir("tests/proxycommands/testdata/proxycommands-no-usr-local")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("no-usr-local")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		// create a new dev command and start it
		done := make(chan error)
		cancelCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

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

		// Check that the command is proxied to the host.
		var stdout, stderr bytes.Buffer
		cmd := exec.Command("helm", "version")
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err = cmd.Run()
		framework.ExpectNoError(err)

		framework.ExpectRemoteFileContents("busybox", ns, "helm-version.out", stdout.String())

		cancel()

		err = <-done
		framework.ExpectNoError(err)
	})
})
