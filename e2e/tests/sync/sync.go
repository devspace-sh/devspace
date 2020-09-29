package sync

import (
	"time"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

type customFactory struct {
	*utils.BaseCustomFactory
	interrupt chan error
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

func (s *fakeServiceClient) StartSyncFromCmd(syncConfig *latest.SyncConfig, interrupt chan error, verbose bool) error {
	err := s.Client.StartSyncFromCmd(syncConfig, s.factory.interrupt, verbose)
	return err
}

func (s *fakeServiceClient) StartSync(interrupt chan error, printSync, verboseSync bool) error {
	err := s.Client.StartSync(s.factory.interrupt, printSync, verboseSync)
	return err
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
	"default":       runDefault,
	"download-only": runDownloadOnly,
	"upload-only":   runUploadOnly,
}

func (r *Runner) Run(subTests []string, ns string, pwd string, logger log.Logger, verbose bool, timeout int) error {
	logger.Info("Run 'sync' test")

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
				f.Namespace = utils.GenerateNamespaceName("test-sync-" + subTestName)

				err := beforeTest(f)
				defer afterTest(f)
				if err != nil {
					return errors.Errorf("test 'sync' failed: %s %v", f.GetLogContents(), err)
				}

				err = availableSubTests[subTestName](f, logger)
				utils.PrintTestResult("sync", subTestName, err, logger)
				if err != nil {
					return errors.Errorf("test 'sync' failed: %s %v", f.GetLogContents(), err)
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

func beforeTest(f *customFactory) error {
	deployConfig := &cmd.DeployCmd{
		GlobalFlags: &flags.GlobalFlags{
			Namespace: f.Namespace,
			NoWarn:    true,
		},
		ForceBuild:  false,
		ForceDeploy: false,
		SkipPush:    true,
	}

	dirPath, _, err := utils.CreateTempDir()
	if err != nil {
		return err
	}

	f.DirPath = dirPath

	err = utils.Copy(f.Pwd+"/tests/sync/testdata", dirPath)
	if err != nil {
		return err
	}

	err = utils.ChangeWorkingDir(dirPath+"/test", f.GetLog())
	if err != nil {
		return err
	}

	// Create kubectl client
	client, err := f.NewKubeClientFromContext(deployConfig.KubeContext, deployConfig.Namespace, deployConfig.SwitchContext)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	f.Client = client

	err = deployConfig.Run(f, nil,nil, nil)
	if err != nil {
		return err
	}

	time.Sleep(time.Second * 5)

	// Checking if pods are running correctly
	err = utils.AnalyzePods(client, f.Namespace, f.GetLog())
	if err != nil {
		return err
	}

	return nil
}

func afterTest(f *customFactory) {
	utils.DeleteTempAndResetWorkingDir(f.DirPath, f.Pwd, f.GetLog())
	utils.DeleteNamespace(f.Client, f.Namespace)
}
