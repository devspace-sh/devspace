package imports

import (
	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/onsi/ginkgo"
	"os"
)

var _ = DevSpaceDescribe("imports", func() {
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

	ginkgo.It("should import correctly", func() {
		tempDir, err := framework.CopyToTempDir("tests/imports/testdata/local")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("imports")
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
		framework.ExpectLocalFileContentsWithoutSpaces("dependency.txt", "import3")
		framework.ExpectLocalFileContentsWithoutSpaces("import1.txt", "import1")
		framework.ExpectLocalFileContentsWithoutSpaces("import2.txt", "import2")
		framework.ExpectLocalFileContentsWithoutSpaces("import3.txt", "import3")
		framework.ExpectLocalFileContentsWithoutSpaces("vars.txt", ns+"-"+ns+"-base-import1-import2-import3")
	})
})
