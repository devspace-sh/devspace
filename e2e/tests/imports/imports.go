package imports

import (
	"bytes"
	"os"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/onsi/ginkgo/v2"
	"gopkg.in/yaml.v3"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/util/factory"
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

	ginkgo.It("should import correctly with variables", func() {
		tempDir, err := framework.CopyToTempDir("tests/imports/testdata/conditional")
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
				Profiles:  []string{},
			},
			Pipeline: "deploy",
		}

		// run the command
		err = deployCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// read temp folder
		framework.ExpectLocalFileContentsWithoutSpaces("import1.txt", "import1")

		// change path
		err = os.Setenv("IMPORT1_PATH", "import2.yaml")
		framework.ExpectNoError(err)
		defer func() {
			_ = os.Unsetenv("IMPORT1_PATH")
		}()

		// create a new dev command
		deployCmd = &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
				Profiles:  []string{},
			},
			Pipeline: "deploy",
		}

		// run the command
		err = deployCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// read temp folder
		framework.ExpectLocalFileContentsWithoutSpaces("import1.txt", "import2")
		framework.ExpectLocalFileContentsWithoutSpaces("message.txt", "Life is Good!")
		framework.ExpectLocalFileContentsWithoutSpaces("hello.txt", "")
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
				Profiles:  []string{"my-profile"},
			},
			Pipeline: "deploy",
		}

		// run the command
		err = deployCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// read temp folder
		out, err := os.ReadFile("temp.txt")
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
		framework.ExpectLocalFileContentsWithoutSpaces("profile_import.txt", "PROFILE_TEST")
		framework.ExpectLocalFileContentsWithoutSpaces("vars.txt", ns+"-"+ns+"-base-import1-import2-import3")
		framework.ExpectLocalFileContentsWithoutSpaces("top.txt", "top")

		// make sure temp folder is erased
		_, err = os.Stat(strings.TrimSpace(string(out)))
		framework.ExpectError(err)
	})

	ginkgo.It("should import correctly with localRegistry", func() {
		tempDir, err := framework.CopyToTempDir("tests/imports/testdata/localregistry")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{},
			Out:         configBuffer,
			SkipInfo:    true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate config
		framework.ExpectEqual(*latestConfig.LocalRegistry.Enabled, false)
		framework.ExpectEqual(latestConfig.LocalRegistry.Name, "defaults-registry")
	})
})
