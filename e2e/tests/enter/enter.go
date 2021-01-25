package enter

import (
	"time"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/utils"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

type Runner struct{}

var RunNew = &Runner{}

func (r *Runner) SubTests() []string {
	subTests := []string{}
	for k := range availableSubTests {
		subTests = append(subTests, k)
	}

	return subTests
}

var availableSubTests = map[string]func(factory *utils.BaseCustomFactory, logger log.Logger) error{
	"default": runDefault,
}

func (r *Runner) Run(subTests []string, ns string, pwd string, logger log.Logger, verbose bool, timeout int) error {
	logger.Info("Run test 'enter'")

	// Populates the tests to run with all the available sub tests if no sub tests are specified
	if len(subTests) == 0 {
		for subTestName := range availableSubTests {
			subTests = append(subTests, subTestName)
		}
	}

	f := &utils.BaseCustomFactory{
		Pwd:     pwd,
		Verbose: verbose,
		Timeout: timeout,
	}

	// Runs the tests
	for _, subTestName := range subTests {
		f.ResetLog()
		c1 := make(chan error)

		go func() {
			err := func() error {
				f.Namespace = utils.GenerateNamespaceName("test-enter-" + subTestName)

				err := beforeTest(f)
				defer afterTest(f)
				if err != nil {
					return errors.Errorf("test 'enter' failed: %s %v", f.GetLogContents(), err)
				}

				err = availableSubTests[subTestName](f, logger)
				utils.PrintTestResult("enter", subTestName, err, logger)
				if err != nil {
					return errors.Errorf("test 'enter' failed: %s %v", f.GetLogContents(), err)
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

func beforeTest(f *utils.BaseCustomFactory) error {
	deployConfig := &cmd.DeployCmd{
		GlobalFlags: &flags.GlobalFlags{
			Namespace: f.Namespace,
			NoWarn:    true,
		},
		ForceBuild: false,
		SkipBuild:  true,
		SkipPush:   true,
	}

	dirPath, _, err := utils.CreateTempDir()
	if err != nil {
		return err
	}

	f.DirPath = dirPath

	err = utils.Copy(f.Pwd+"/tests/enter/testdata", dirPath)
	if err != nil {
		return err
	}

	err = utils.ChangeWorkingDir(dirPath, f.GetLog())
	if err != nil {
		return err
	}

	// Create kubectl client
	client, err := f.NewKubeClientFromContext(deployConfig.KubeContext, deployConfig.Namespace, deployConfig.SwitchContext)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	f.Client = client
	err = deployConfig.Run(f, nil, nil, nil)
	if err != nil {
		return errors.Errorf("An error occured while deploying: %v", err)
	}

	time.Sleep(5 * time.Second)

	// Checking if pods are running correctly
	err = utils.AnalyzePods(client, f.Namespace, f.GetLog())
	if err != nil {
		return errors.Errorf("An error occured while analyzing pods: %v", err)
	}

	return nil
}

func afterTest(f *utils.BaseCustomFactory) {
	utils.DeleteTempAndResetWorkingDir(f.DirPath, f.Pwd, f.GetLog())
	utils.DeleteNamespace(f.Client, f.Namespace)
}
