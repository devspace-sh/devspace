package initcmd

import (
	"bytes"
	"path/filepath"
	"reflect"
	"time"

	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/sirupsen/logrus"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/e2e/utils"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/helper"
	"github.com/loft-sh/devspace/pkg/devspace/configure"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	fakesurvey "github.com/loft-sh/devspace/pkg/util/survey/testing"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	fakedocker "github.com/loft-sh/devspace/pkg/devspace/docker/testing"
	"github.com/pkg/errors"

	yaml "gopkg.in/yaml.v2"
)

type initTestCase struct {
	name    string
	answers []string

	expectedConfig *latest.Config
	tempLogger     log.Logger
}

type customFactory struct {
	*utils.BaseCustomFactory
}

type customLogger struct {
	log.Logger
	*fakesurvey.FakeSurvey
}

func (c *customFactory) NewDockerClientWithMinikube(currentKubeContext string, preferMinikube bool, log log.Logger) (docker.Client, error) {
	fakeDockerClient := &fakedocker.FakeClient{
		AuthConfig: &dockertypes.AuthConfig{
			Username: "user",
			Password: "pass",
		},
	}
	return fakeDockerClient, nil
}

func (c *customFactory) NewConfigureManager(config *latest.Config, log log.Logger) configure.Manager {
	return configure.NewManager(c, config, log)
}

func (c *customLogger) Question(params *survey.QuestionOptions) (string, error) {
	return c.FakeSurvey.Question(params)
}

// GetLog implements interface
func (c *customFactory) GetLog() log.Logger {
	if c.CacheLogger == nil {
		if c.Verbose {
			c.CacheLogger = &customLogger{
				Logger:     log.GetInstance(),
				FakeSurvey: fakesurvey.NewFakeSurvey(),
			}
		} else {
			c.Buff = &bytes.Buffer{}
			c.CacheLogger = &customLogger{
				Logger:     log.NewStreamLogger(c.Buff, logrus.InfoLevel),
				FakeSurvey: fakesurvey.NewFakeSurvey(),
			}
		}
	}

	return c.CacheLogger
}

var availableSubTests = map[string]func(factory *customFactory, logger log.Logger) error{
	"create_dockerfile":       CreateDockerfile,
	"use_existing_dockerfile": UseExistingDockerfile,
	"use_dockerfile":          UseDockerfile,
	"use_manifests":           UseManifests,
	"use_chart":               UseChart,
}

type Runner struct{}

var RunNew = &Runner{}

func (r *Runner) SubTests() []string {
	subTests := []string{}
	for k := range availableSubTests {
		subTests = append(subTests, k)
	}

	return subTests
}

func (r *Runner) Run(subTests []string, ns string, pwd string, logger log.Logger, verbose bool, timeout int) error {
	logger.Info("Run 'init' test")

	// Populates the tests to run with all the available sub tests if no sub tests is specified
	if len(subTests) == 0 {
		for subTestName := range availableSubTests {
			subTests = append(subTests, subTestName)
		}
	}

	f := &customFactory{
		BaseCustomFactory: &utils.BaseCustomFactory{
			Pwd:     pwd,
			Verbose: verbose,
			Timeout: timeout,
		},
	}

	// Runs the tests
	for _, subTestName := range subTests {
		f.ResetLog()
		c1 := make(chan error)

		go func() {
			err := func() error {
				f.Namespace = utils.GenerateNamespaceName("test-init-" + subTestName)
				err := availableSubTests[subTestName](f, logger)
				utils.PrintTestResult("init", subTestName, err, logger)
				if err != nil {
					return errors.Errorf("test 'init' failed: %s %v", f.GetLogContents(), err)
				}

				return nil
			}()
			c1 <- err
		}()

		select {
		case err := <-c1:
			if err != nil {
				return err
			}
		case <-time.After(time.Duration(timeout) * time.Second):
			return errors.Errorf("Timeout error - the test did not return within the specified timeout of %v seconds: %s", timeout, f.GetLogContents())
		}
	}

	return nil
}

func runTest(f *customFactory, testCase initTestCase) error {
	initConfig := cmd.InitCmd{
		Dockerfile:  helper.DefaultDockerfilePath,
		Reconfigure: false,
		Context:     "",
		Provider:    "",
	}

	for _, a := range testCase.answers {
		f.GetLog().(*customLogger).SetNextAnswer(a)
	}

	// runs init cmd
	err := initConfig.Run(f, nil, nil)
	if err != nil {
		return err
	}

	if testCase.expectedConfig != nil {
		config, err := f.NewConfigLoader(nil, nil).Load()
		if err != nil {
			return err
		}

		isEqual := reflect.DeepEqual(config, testCase.expectedConfig)
		if !isEqual {
			configYaml, _ := yaml.Marshal(config)
			expectedYaml, _ := yaml.Marshal(testCase.expectedConfig)

			return errors.Errorf("TestCase '%v': Got\n %s\n\n, but expected\n\n %s\n", testCase.name, configYaml, expectedYaml)
		}
	}

	return nil
}

func beforeTest(f *customFactory, logger log.Logger, testDir string) error {
	testDir = filepath.FromSlash(testDir)

	dirPath, dirName, err := utils.CreateTempDir()
	if err != nil {
		return err
	}

	f.DirPath = dirPath
	f.DirName = dirName

	// Copy the testdata into the temp dir
	err = utils.Copy(testDir, dirPath)
	if err != nil {
		return err
	}

	// Change working directory
	err = utils.ChangeWorkingDir(dirPath, f.GetLog())
	if err != nil {
		return err
	}

	return nil
}

func afterTest(f *customFactory) {
	utils.DeleteTempAndResetWorkingDir(f.DirPath, f.Pwd, f.GetLog())
}
