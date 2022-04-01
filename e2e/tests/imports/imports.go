package imports

import (
	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/onsi/ginkgo"
	"io/ioutil"
	"os"
	"strings"
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
		deployCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Pipeline: "deploy",
		}

		// run the command
		err = deployCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// read temp folder
		out, err := ioutil.ReadFile("temp.txt")
		framework.ExpectNoError(err)
		framework.ExpectLocalFileContentsWithoutSpaces("name.txt", "base")
		framework.ExpectLocalFileContentsWithoutSpaces("dependency.txt", "import3")
		framework.ExpectLocalFileContentsWithoutSpaces("dependency-name.txt", "import1")
		framework.ExpectLocalFileContentsWithoutSpaces("dependency-temp.txt", strings.TrimSpace(string(out)))
		framework.ExpectLocalFileContentsWithoutSpaces("import1.txt", "import1")
		framework.ExpectLocalFileContentsWithoutSpaces("import2.txt", "import2")
		framework.ExpectLocalFileContentsWithoutSpaces("import2-name.txt", "base")
		framework.ExpectLocalFileContentsWithoutSpaces("import3.txt", "import3")
		framework.ExpectLocalFileContentsWithoutSpaces("import4.txt", "import4")
		framework.ExpectLocalFileContentsWithoutSpaces("import5.txt", "import5")
		framework.ExpectLocalFileContentsWithoutSpaces("vars.txt", ns+"-"+ns+"-base-import1-import2-import3")

		// make sure temp folder is erased
		_, err = os.Stat(strings.TrimSpace(string(out)))
		framework.ExpectError(err)
	})
})
