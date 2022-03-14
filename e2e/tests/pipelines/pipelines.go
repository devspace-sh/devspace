package pipelines

import (
	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/onsi/ginkgo"
	"os"
)

var _ = DevSpaceDescribe("portforward", func() {
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

	ginkgo.It("should exec container", func() {
		tempDir, err := framework.CopyToTempDir("tests/pipelines/testdata/exec_container")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("pipelines")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		devCmd := &cmd.DevCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
		}
		err = devCmd.Run(f)
		framework.ExpectNoError(err)
		framework.ExpectLocalFileContentsImmediately("test.txt", "Hello World!\n")
	})
})
