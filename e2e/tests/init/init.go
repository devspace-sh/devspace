package init

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/onsi/ginkgo"
)

var _ = DevSpaceDescribe("init", func() {
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

	ginkgo.It("should create devspace.yml without registry details", func() {
		tempDir, err := framework.CopyToTempDir("tests/init/testdata/new")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// set the question answer func here
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			if strings.Contains(params.Question, "Which registry would you want to use to push images to?") {
				return "Skip Registry", nil
			}

			return params.DefaultValue, nil
		})

		initCmd := &cmd.InitCmd{}
		err = initCmd.Run(f)
		framework.ExpectNoError(err)

		config, _, err := framework.LoadConfig(f, filepath.Join(tempDir, "devspace.yaml"))
		framework.ExpectNoError(err)

		framework.ExpectEqual(config.Variables(), map[string]interface{}{"IMAGE": "username/app"})
	})

	ginkgo.It("should create devspace.yml from docker-compose.yaml", func() {
		tempDir, err := framework.CopyToTempDir("tests/init/testdata/docker-compose")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// Answer all questions with the default
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			fmt.Println(params.Question)
			return params.DefaultValue, nil
		})

		initCmd := &cmd.InitCmd{
			Reconfigure: true,
		}
		err = initCmd.Run(f)
		framework.ExpectNoError(err)

		// Created a devspace.yaml
		_, _, err = framework.LoadConfig(f, filepath.Join(tempDir, "devspace.yaml"))
		framework.ExpectNoError(err)

		// Created a .gitignore
		_, err = os.Stat(filepath.Join(tempDir, ".gitignore"))
		framework.ExpectNoError(err)

		// Created a .devspace/generated.yaml
		_, err = os.Stat(filepath.Join(tempDir, ".devspace", "generated.yaml"))
		framework.ExpectNoError(err)
	})
})
