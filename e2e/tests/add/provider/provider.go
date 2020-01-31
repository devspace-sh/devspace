package provider

import (
	"time"

	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	fakecloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/testing"
	cloudconfiglatest "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

type customFactory struct {
	*utils.BaseCustomFactory
}

func (c *customFactory) NewCloudConfigLoader() config.Loader {
	return fakecloudconfig.NewLoader(&cloudconfiglatest.Config{
		Version: cloudconfiglatest.Version,
		Default: "test-provider",
	})
}

// GetProviderWithOptions implements interface
func (f *customFactory) GetProviderWithOptions(useProviderName, key string, relogin bool, loader config.Loader, kubeLoader kubeconfig.Loader, log log.Logger) (cloud.Provider, error) {
	return nil, nil
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
	logger.Info("Run test 'add-provider'")

	// Populates the tests to run with all the available sub tests if no sub tests are specified
	if len(subTests) == 0 {
		for subTestName := range availableSubTests {
			subTests = append(subTests, subTestName)
		}
	}

	f := &customFactory{
		&utils.BaseCustomFactory{
			Pwd:     pwd,
			Verbose: verbose,
			Timeout: timeout,
		},
	}

	// Runs the tests
	for _, subTestName := range subTests {
		f.ResetLog()
		c1 := make(chan error, 1)

		go func() {
			err := func() error {
				f.Namespace = utils.GenerateNamespaceName("test-add-provider-" + subTestName)

				defer afterTest(f)

				err := availableSubTests[subTestName](f, logger)
				utils.PrintTestResult("add-provider", subTestName, err, logger)
				if err != nil {
					return errors.Errorf("test 'add-provider' failed: %s %v", f.GetLogContents(), err)
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

func afterTest(f *customFactory) {
	// utils.DeleteTempAndResetWorkingDir(f.DirPath, f.Pwd, f.GetLog())
	// utils.DeleteNamespace(f.Client, f.Namespace)
}
