package dependencies

import (
	"os"
	"path/filepath"

	"github.com/loft-sh/devspace/e2e/new/framework"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/onsi/ginkgo"
)

var _ = DevSpaceDescribe("dependencies", func() {
	initialDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// create a new factory
	var (
		f *framework.DefaultFactory
	)

	ginkgo.BeforeEach(func() {
		f = framework.NewDefaultFactory()
	})

	ginkgo.It("should resolve dependencies with local path and nested structure", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/nested")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// set the question answer func here
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			return "nested", nil
		})

		// load it from the regular path first
		_, dependencies, err := framework.LoadConfig(f, filepath.Join(tempDir, "devspace.yaml"))
		framework.ExpectNoError(err)

		// check if dependencies were loaded correctly
		framework.ExpectEqual(len(dependencies), 1)
		framework.ExpectEqual(dependencies[0].Name(), "nested")
	})

	ginkgo.It("should resolve dependencies with local path and flat structure", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/flat")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// set the question answer func here
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			return "flat", nil
		})

		// load it from the regular path first
		_, dependencies, err := framework.LoadConfig(f, filepath.Join(tempDir, "devspace.yaml"))
		framework.ExpectNoError(err)

		// check if dependencies were loaded correctly
		framework.ExpectEqual(len(dependencies), 1)
		framework.ExpectEqual(dependencies[0].Name(), "flat")
	})

	ginkgo.It("should resolve dependencies and activate dependency profiles", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/profile-activation")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// load it from the regular path first
		os.Setenv("FOO", "true")
		defer os.Unsetenv("FOO")
		_, dependencies, err := framework.LoadConfig(f, filepath.Join(tempDir, "activated.yaml"))
		framework.ExpectNoError(err)

		// check if dependencies were loaded correctly with profile activation
		framework.ExpectEqual(len(dependencies), 1)
		framework.ExpectEqual(dependencies[0].Name(), "nested")
		framework.ExpectEqual(len(dependencies[0].Config().Config().Deployments), 2)
	})

	ginkgo.It("should resolve dependencies and deactivate dependency profiles", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/profile-activation")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// load it from the regular path first
		os.Setenv("FOO", "true")
		defer os.Unsetenv("FOO")
		_, dependencies, err := framework.LoadConfig(f, filepath.Join(tempDir, "deactivated.yaml"))
		framework.ExpectNoError(err)

		// check if dependencies were loaded correctly without profile activation
		framework.ExpectEqual(len(dependencies), 1)
		framework.ExpectEqual(dependencies[0].Name(), "nested")
		framework.ExpectEqual(len(dependencies[0].Config().Config().Deployments), 1)
	})
})
