package enter

import (
	"bytes"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type customFactory struct {
	*factory.DefaultFactoryImpl
	verbose     bool
	timeout     int
	namespace   string
	pwd         string
	cacheLogger log.Logger
	dirPath     string
	client      kubectl.Client
}

func (cf *customFactory) GetLog() log.Logger {
	return cf.cacheLogger
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

var availableSubTests = map[string]func(factory *customFactory, logger log.Logger) error{
	"default": runDefault,
}

func (r *Runner) Run(subTests []string, ns string, pwd string, logger log.Logger, verbose bool, timeout int) error {
	buff := &bytes.Buffer{}
	var cacheLogger log.Logger
	cacheLogger = log.NewStreamLogger(buff, logrus.InfoLevel)

	var buffString string
	buffString = buff.String()

	if verbose {
		cacheLogger = logger
		buffString = ""
	}

	logger.Info("Run test 'enter'")

	// Populates the tests to run with all the available sub tests if no sub tests are specified
	if len(subTests) == 0 {
		for subTestName := range availableSubTests {
			subTests = append(subTests, subTestName)
		}
	}

	f := &customFactory{
		pwd:         pwd,
		cacheLogger: cacheLogger,
		verbose:     verbose,
		timeout:     timeout,
	}

	// Runs the tests
	for _, subTestName := range subTests {
		f.namespace = utils.GenerateNamespaceName("test-enter-" + subTestName)

		err := beforeTest(f)
		defer afterTest(f)
		if err != nil {
			return errors.Errorf("test 'enter' failed: %s %v", buffString, err)
		}

		err = availableSubTests[subTestName](f, logger)
		utils.PrintTestResult("enter", subTestName, err, logger)
		if err != nil {
			return errors.Errorf("test 'enter' failed: %s %v", buffString, err)
		}
	}

	return nil
}

func beforeTest(f *customFactory) error {
	deployConfig := &cmd.DeployCmd{
		GlobalFlags: &flags.GlobalFlags{
			Namespace: f.namespace,
			NoWarn:    true,
		},
	}

	dirPath, _, err := utils.CreateTempDir()
	if err != nil {
		return err
	}

	f.dirPath = dirPath

	err = utils.Copy(f.pwd+"/tests/enter/testdata", dirPath)
	if err != nil {
		return err
	}

	err = utils.ChangeWorkingDir(dirPath, f.cacheLogger)
	if err != nil {
		return err
	}

	// Create kubectl client
	client, err := f.NewKubeDefaultClient()
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	f.client = client

	err = deployConfig.Run(f, nil, nil)
	if err != nil {
		return errors.Errorf("An error occured while deploying: %v", err)
	}

	// Checking if pods are running correctly
	err = utils.AnalyzePods(client, f.namespace, f.cacheLogger)
	if err != nil {
		return errors.Errorf("An error occured while analyzing pods: %v", err)
	}

	return nil
}

func afterTest(f *customFactory) {
	utils.DeleteTempAndResetWorkingDir(f.dirPath, f.pwd, f.cacheLogger)
	utils.DeleteNamespace(f.client, f.namespace)
}
