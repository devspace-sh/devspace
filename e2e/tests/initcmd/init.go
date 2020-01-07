package initcmd

import (
	"bytes"
	"io"
	"path/filepath"
	"reflect"

	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/sirupsen/logrus"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/devspace/build/builder/helper"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	fakesurvey "github.com/devspace-cloud/devspace/pkg/util/survey/testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/pkg/errors"

	yaml "gopkg.in/yaml.v2"
)

type initTestCase struct {
	name    string
	answers []string

	expectedConfig *latest.Config
}

type customFactory struct {
	*factory.DefaultFactoryImpl
	namespace   string
	pwd         string
	cacheLogger *customLogger
	dirPath     string
	dirName     string
	client      kubectl.Client
}

type customLogger struct {
	*log.StreamLogger
	*fakesurvey.FakeSurvey
}

func NewCustomStreamLogger(stream io.Writer, level logrus.Level) *customLogger {
	return &customLogger{
		StreamLogger: log.NewStreamLogger(stream, level),
		FakeSurvey:   fakesurvey.NewFakeSurvey(),
	}
}

func (c *customLogger) Question(params *survey.QuestionOptions) (string, error) {
	return c.FakeSurvey.Question(params)
}

// func (c *customFactory) NewConfigLoader(options *loader.ConfigOptions, log log.Logger) loader.ConfigLoader {
// 	return fakeconfigloader.NewFakeConfigLoader(c.GeneratedConfig, c.Config, log)
// }

// NewDockerClient implements interface
func (c *customFactory) NewDockerClient(log log.Logger) (docker.Client, error) {
	fakeDockerClient := &docker.FakeClient{
		AuthConfig: &dockertypes.AuthConfig{
			Username: "user",
			Password: "pass",
		},
	}
	return fakeDockerClient, nil
}

// GetLog implements interface
func (c *customFactory) GetLog() log.Logger {
	return c.cacheLogger
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

func (r *Runner) Run(subTests []string, ns string, pwd string, logger log.Logger) error {
	buff := &bytes.Buffer{}

	logger.Info("Run 'init' test")

	// Populates the tests to run with all the available sub tests if no sub tests is specified
	if len(subTests) == 0 {
		for subTestName := range availableSubTests {
			subTests = append(subTests, subTestName)
		}
	}

	f := &customFactory{
		pwd: pwd,
	}

	// Runs the tests
	for _, subTestName := range subTests {
		f.namespace = utils.GenerateNamespaceName("test-init-" + subTestName)
		err := availableSubTests[subTestName](f, logger)
		utils.PrintTestResult("init", subTestName, err, logger)
		if err != nil {
			return errors.Errorf("test 'init' failed: %s %v", buff.String(), err)
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

	c, err := f.NewDockerClient(f.cacheLogger)
	if err != nil {
		return err
	}
	docker.SetFakeClient(c)

	for _, a := range testCase.answers {
		// fmt.Println("SetNextAnswer:", a)
		f.cacheLogger.SetNextAnswer(a)
	}

	// runs init cmd
	err = initConfig.Run(f, nil, nil)
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

	f.dirPath = dirPath
	f.dirName = dirName

	// Copy the testdata into the temp dir
	err = utils.Copy(testDir, dirPath)
	if err != nil {
		return err
	}

	// Change working directory
	err = utils.ChangeWorkingDir(dirPath, f.cacheLogger)
	if err != nil {
		return err
	}

	return nil
}

func afterTest(f *customFactory) {
	utils.DeleteTempAndResetWorkingDir(f.dirPath, f.pwd, f.cacheLogger)
	utils.DeleteNamespace(f.client, f.namespace)
}
