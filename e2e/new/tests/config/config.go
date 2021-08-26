package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/loft-sh/devspace/cmd/use"
	"github.com/loft-sh/devspace/e2e/new/framework"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/onsi/ginkgo"
)

var _ = DevSpaceDescribe("config", func() {
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

	ginkgo.It("should load profile cached and uncached", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// set the question answer func here
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			return "test", nil
		})

		// load it without profile
		config, _, err := framework.LoadConfig(f, "devspace.yaml")
		framework.ExpectNoError(err)

		// check no profile was loaded
		framework.ExpectEqual(len(config.Config().Images), 1)
		framework.ExpectEqual(len(config.Config().Deployments), 1)

		// now set the profile via command
		profileCmd := &use.ProfileCmd{}

		// try to set non existing profile
		err = profileCmd.RunUseProfile(f, nil, []string{"does-not-exist"})
		framework.ExpectError(err)

		// set profile correctly
		err = profileCmd.RunUseProfile(f, nil, []string{"remove-image"})
		framework.ExpectNoError(err)

		// reload it
		config, _, err = framework.LoadConfig(f, "devspace.yaml")
		framework.ExpectNoError(err)

		// check profile was loaded
		framework.ExpectEqual(len(config.Config().Images), 0)
		framework.ExpectEqual(len(config.Config().Deployments), 1)

		// reload it and set it through config options
		config, _, err = framework.LoadConfigWithOptions(f, "devspace.yaml", &loader.ConfigOptions{Profile: "add-deployment"})
		framework.ExpectNoError(err)

		// check profile was loaded
		framework.ExpectEqual(len(config.Config().Images), 1)
		framework.ExpectEqual(len(config.Config().Deployments), 2)
	})

	ginkgo.It("should resolve variables correctly", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/vars")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// set the question answer func here
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			return "test", nil
		})

		// load it from the regular path first
		config, dependencies, err := framework.LoadConfig(f, filepath.Join(tempDir, "devspace.yaml"))
		framework.ExpectNoError(err)

		// check if variables were loaded correctly
		framework.ExpectEqual(len(config.Variables()), 4)
		framework.ExpectEqual(len(config.Generated().Vars), 1)
		framework.ExpectEqual(config.Generated().Vars["TEST_1"], "test")
		framework.ExpectEqual(len(dependencies), 1)
		framework.ExpectEqual(len(dependencies[0].Config().Generated().Vars), 1)
		framework.ExpectEqual(dependencies[0].Config().Generated().Vars["NOT_USED"], "test")
		framework.ExpectEqual(dependencies[0].Config().Variables()["TEST_OVERRIDE"], "devspace.yaml")

		// make sure we don't get asked again
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			return "", fmt.Errorf("shouldn't get asked again")
		})

		// rerun now with cached
		_, _, err = framework.LoadConfig(f, filepath.Join(tempDir, "devspace.yaml"))
		framework.ExpectNoError(err)

		// make sure we don't get asked again
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			return "dep1", nil
		})

		// rerun now with cached
		config, dependencies, err = framework.LoadConfig(f, filepath.Join(tempDir, "dep1", "dev.yaml"))
		framework.ExpectNoError(err)

		// config
		framework.ExpectEqual(len(config.Variables()), 3)
		framework.ExpectEqual(len(config.Generated().Vars), 2)
		framework.ExpectEqual(config.Generated().Vars["NOT_USED"], "test")
		framework.ExpectEqual(config.Generated().Vars["TEST_2"], "dep1")
		framework.ExpectEqual(config.Variables()["TEST_OVERRIDE"], "dev.yaml")
		framework.ExpectEqual(len(dependencies), 0)
	})

	ginkgo.It("should cache multiple configs independently", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/multiple")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// set the question answer func here
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			return "default", nil
		})

		// load it from the default path
		config, dependencies, err := framework.LoadConfig(f, filepath.Join(tempDir, "devspace.yaml"))
		framework.ExpectNoError(err)

		// check if default config variables were loaded correctly
		framework.ExpectEqual(len(config.Variables()), 2)
		framework.ExpectEqual(len(config.Generated().Vars), 1)
		framework.ExpectEqual(config.Generated().Vars["NAME"], "default")
		framework.ExpectEqual(len(dependencies), 0)

		// set the question answer func here
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			return "custom", nil
		})

		// load it from a custom path
		customConfig, customDependencies, err := framework.LoadConfig(f, filepath.Join(tempDir, "custom.yaml"))
		framework.ExpectNoError(err)

		// check if custom config variables were loaded correctly
		framework.ExpectEqual(len(customConfig.Variables()), 2)
		framework.ExpectEqual(len(customConfig.Generated().Vars), 1)
		framework.ExpectEqual(customConfig.Generated().Vars["NAME"], "custom")
		framework.ExpectEqual(len(customDependencies), 0)

		// make sure we don't get asked again
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			return "", fmt.Errorf("shouldn't get asked again")
		})

		// reload default config with cache
		_, _, err = framework.LoadConfig(f, filepath.Join(tempDir, "devspace.yaml"))
		framework.ExpectNoError(err)

		// make sure we don't get asked again
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			return "", fmt.Errorf("shouldn't get asked again")
		})

		// reload custom config with cache
		_, _, err = framework.LoadConfig(f, filepath.Join(tempDir, "custom.yaml"))
		framework.ExpectNoError(err)
	})
})
