package dev

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"

	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
)

type customFactory struct {
	*utils.BaseCustomFactory
	initialRun            bool
	imageSelectorFirstRun string
	interrupt             chan error
}

type fakeServiceClient struct {
	services.Client
	factory           *customFactory
	selectorParameter *targetselector.SelectorParameter
}

// NewFakeServiceClient implements
func (c *customFactory) NewServicesClient(config *latest.Config, generated *generated.Config, kubeClient kubectl.Client, selectorParameter *targetselector.SelectorParameter, log log.Logger) services.Client {
	c.interrupt = make(chan error)

	return &fakeServiceClient{
		Client:            services.NewClient(config, generated, kubeClient, selectorParameter, log),
		factory:           c,
		selectorParameter: selectorParameter,
	}
}

func (s *fakeServiceClient) StartPortForwarding(interrupt chan error) error {
	err := s.Client.StartPortForwarding(s.factory.interrupt)
	return err
}

func (s *fakeServiceClient) StartSync(interrupt chan error, verboseSync bool) error {
	err := s.Client.StartSync(s.factory.interrupt, verboseSync)
	return err
}

func (serviceClient *fakeServiceClient) StartTerminal(args []string, imageSelector []string, interrupt chan error, wait bool) (int, error) {
	if !serviceClient.factory.initialRun {
		serviceClient.factory.initialRun = true
		serviceClient.factory.imageSelectorFirstRun = imageSelector[0]

		// Portforwarding test
		err := checkPortForwarding(serviceClient.factory)
		if err != nil {
			return 0, errors.Errorf("Check portforwarding failed: %v", err)
		}

		// Deletes a sync file to trigger the autoReload
		err = os.Remove(filepath.FromSlash(serviceClient.factory.DirPath + "/foo/bar.go"))
		if err != nil {
			return 0, err
		}

		return 0, <-interrupt
	}

	// autoReload + sync test
	if serviceClient.factory.imageSelectorFirstRun == imageSelector[0] {
		return 0, errors.New("autoReload error: image failed to rebuild (image selectors should be different, but were found to be similar)")
	}

	return 0, nil
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
	"default": RunDefault,
}

func (r *Runner) Run(subTests []string, ns string, pwd string, logger log.Logger, verbose bool, timeout int) error {
	logger.Info("Run 'dev' test")

	// Populates the tests to run with all the available sub tests if no sub tests are specified
	if len(subTests) == 0 {
		for subTestName := range availableSubTests {
			subTests = append(subTests, subTestName)
		}
	}

	f := &customFactory{
		BaseCustomFactory: &utils.BaseCustomFactory{
			Namespace: ns,
			Pwd:       pwd,
			Verbose:   verbose,
			Timeout:   timeout,
		},
	}

	// Runs the tests
	for _, subTestName := range subTests {
		f.ResetLog()
		c1 := make(chan error)

		go func() {
			err := func() error {
				f.Namespace = utils.GenerateNamespaceName("test-dev-" + subTestName)
				err := availableSubTests[subTestName](f, logger)
				utils.PrintTestResult("dev", subTestName, err, logger)
				if err != nil {
					return err
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

func beforeTest(f *customFactory, logger log.Logger, testDir string) error {
	testDir = filepath.FromSlash(testDir)

	dirPath, _, err := utils.CreateTempDir()
	if err != nil {
		return err
	}

	f.DirPath = dirPath

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
	utils.DeleteNamespace(f.Client, f.Namespace)
}

func checkPortForwarding(f *customFactory) error {
	url := "http://localhost:8080/"

	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	if resp.StatusCode == 200 {
		f.GetLog().Donef("Pinging %v: status code 200", url)
	} else {
		return fmt.Errorf("pinging %v: status code %v", url, resp.StatusCode)
	}

	return nil
}
