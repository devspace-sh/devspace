package logs

import (
	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	fakelog "github.com/devspace-cloud/devspace/pkg/util/log/testing"
	"github.com/pkg/errors"
)

type customFactory struct {
	*factory.DefaultFactoryImpl
	namespace  string
	pwd        string
	FakeLogger *fakelog.FakeLogger
}

// GetLog implements interface
func (c *customFactory) GetLog() log.Logger {
	return c.FakeLogger
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

var availableSubTests = map[string]func(factory *customFactory) error{
	"default": runDefault,
}

func (r *Runner) Run(subTests []string, ns string, pwd string) error {
	// Populates the tests to run with all the available sub tests if no sub tests are specified
	if len(subTests) == 0 {
		for subTestName := range availableSubTests {
			subTests = append(subTests, subTestName)
		}
	}

	f := &customFactory{
		namespace: ns,
		pwd:       pwd,
	}
	f.FakeLogger = fakelog.NewFakeLogger()

	deployConfig := &cmd.DeployCmd{
		GlobalFlags: &flags.GlobalFlags{
			Namespace: ns,
			NoWarn:    true,
		},
		ForceBuild:  true,
		ForceDeploy: true,
		SkipPush:    true,
	}

	dirPath, _, err := utils.CreateTempDir()
	if err != nil {
		return err
	}

	err = utils.Copy(f.pwd+"/tests/logs/testdata", dirPath)
	if err != nil {
		return err
	}

	err = utils.ChangeWorkingDir(dirPath)
	if err != nil {
		return err
	}

	defer utils.DeleteTempAndResetWorkingDir(dirPath, f.pwd)

	// Create kubectl client
	client, err := f.NewKubeClientFromContext(deployConfig.KubeContext, deployConfig.Namespace, deployConfig.SwitchContext)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	defer utils.DeleteNamespaceAndWait(client, deployConfig.Namespace)

	err = deployConfig.Run(f, nil, nil)
	if err != nil {
		return err
	}

	// Checking if pods are running correctly
	err = utils.AnalyzePods(client, ns)
	if err != nil {
		return err
	}

	// Runs the tests
	for _, subTestName := range subTests {
		err := availableSubTests[subTestName](f)
		utils.PrintTestResult("logs", subTestName, err)
		if err != nil {
			return err
		}
	}

	return nil
}
