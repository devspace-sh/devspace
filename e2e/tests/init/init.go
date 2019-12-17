package init

import (
	"fmt"
	"reflect"

	"github.com/devspace-cloud/devspace/pkg/util/factory"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/devspace/build/builder/helper"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	fakelog "github.com/devspace-cloud/devspace/pkg/util/log/testing"
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
	namespace  string
	pwd        string
	FakeLogger *fakelog.FakeLogger
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
	return c.FakeLogger
}

var availableSubTests = map[string]func(factory *customFactory) error{
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

func (r *Runner) Run(subTests []string, ns string, pwd string) error {
	// Populates the tests to run with all the available sub tests if no sub tests is specified
	if len(subTests) == 0 {
		for subTestName := range availableSubTests {
			subTests = append(subTests, subTestName)
		}
	}

	myFactory := &customFactory{
		namespace: ns,
		pwd:       pwd,
	}
	myFactory.FakeLogger = fakelog.NewFakeLogger()

	// Runs the tests
	for _, subTestName := range subTests {
		err := availableSubTests[subTestName](myFactory)
		utils.PrintTestResult("init", subTestName, err)
		if err != nil {
			return err
		}
	}

	return nil
}

func initializeTest(f *customFactory, testCase initTestCase) error {
	initConfig := cmd.InitCmd{
		Dockerfile:  helper.DefaultDockerfilePath,
		Reconfigure: false,
		Context:     "",
		Provider:    "",
	}

	c, err := f.NewDockerClient(f.GetLog())
	if err != nil {
		return err
	}
	docker.SetFakeClient(c)

	for _, a := range testCase.answers {
		fmt.Println("SetNextAnswer:", a)
		f.FakeLogger.Survey.SetNextAnswer(a)
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
