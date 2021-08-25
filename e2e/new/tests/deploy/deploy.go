package deploy

import (
	"context"
	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/new/framework"
	"github.com/loft-sh/devspace/e2e/new/kube"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
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

	ginkgo.It("should deploy kustomize application", func() {
		// TODO
	})

	ginkgo.It("should deploy helm application", func() {
		tempDir, err := framework.CopyToTempDir("tests/deploy/testdata/helm")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("deploy")
		framework.ExpectNoError(err)
		defer kubeClient.DeleteNamespace(ns)

		// create a new dev command
		deployCmd := &cmd.DeployCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
		}

		// run the command
		err = deployCmd.Run(f, nil, nil, nil)
		framework.ExpectNoError(err)

		// wait until nginx pod is reachable
		out, err := kubeClient.ExecByImageSelector("nginx", ns, []string{"echo", "-n", "test"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(out, "test")
	})

	ginkgo.It("should deploy kubectl application", func() {
		tempDir, err := framework.CopyToTempDir("tests/deploy/testdata/kubectl")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("deploy")
		framework.ExpectNoError(err)
		defer kubeClient.DeleteNamespace(ns)

		// create a new dev command
		deployCmd := &cmd.DeployCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
		}

		// run the command
		err = deployCmd.Run(f, nil, nil, nil)
		framework.ExpectNoError(err)

		// wait until nginx pod is reachable
		out, err := kubeClient.ExecByImageSelector("nginx", ns, []string{"echo", "-n", "test"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(out, "test")

		// wait until nginx pod is reachable
		out, err = kubeClient.ExecByImageSelector("busybox", ns, []string{"echo", "-n", "test"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(out, "test")

		// check if service is there
		_, err = kubeClient.RawClient().CoreV1().Services(ns).Get(context.TODO(), "webserver-simple-service", metav1.GetOptions{})
		framework.ExpectNoError(err)
	})
})
